package main

import (
	"net"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/miekg/dns"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

type Question struct {
	qname  string
	qtype  string
	qclass string
}

func (q *Question) String() string {
	return q.qname + " " + q.qclass + " " + q.qtype
}

type GODNSHandler struct {
	resolver        *Resolver
	cache, negCache Cache
	hosts           Hosts
	refreshGroup    *singleflight.Group
}

func NewHandler() *GODNSHandler {

	var (
		cacheConfig     CacheSettings
		resolver        *Resolver
		cache, negCache Cache
	)

	resolver = NewResolver(settings.ResolvConfig)

	cacheConfig = settings.Cache
	switch cacheConfig.Backend {
	case "memory":
		cache = &MemoryCache{
			Backend:  make(map[string]Mesg, cacheConfig.Maxcount),
			Expire:   time.Duration(cacheConfig.Expire) * time.Second,
			Maxcount: cacheConfig.Maxcount,
		}
		if cacheConfig.NoNegative {
			negCache = &NoCache{}
		} else {
			negCache = &MemoryCache{
				Backend:  make(map[string]Mesg),
				Expire:   time.Duration(cacheConfig.Expire) * time.Second / 2,
				Maxcount: cacheConfig.Maxcount,
			}
		}
	case "memcache":
		cache = NewMemcachedCache(
			settings.Memcache.Servers,
			int32(cacheConfig.Expire))
		if cacheConfig.NoNegative {
			negCache = &NoCache{}
		} else {
			negCache = NewMemcachedCache(
				settings.Memcache.Servers,
				int32(cacheConfig.Expire/2))
		}
	case "redis":
		cache = NewRedisCache(
			settings.Redis,
			int64(cacheConfig.Expire))
		if cacheConfig.NoNegative {
			negCache = &NoCache{}
		} else {
			negCache = NewRedisCache(
				settings.Redis,
				int64(cacheConfig.Expire/2))
		}
	default:
		logger.Error("Invalid cache backend %s", cacheConfig.Backend)
		panic("Invalid cache backend")
	}

	var hosts Hosts
	if settings.Hosts.Enable {
		hosts = NewHosts(settings.Hosts, settings.Redis)
	}

	return &GODNSHandler{resolver, cache, negCache, hosts, &singleflight.Group{}}
}

func (h *GODNSHandler) do(Net string, w dns.ResponseWriter, req *dns.Msg) {
	q := req.Question[0]
	Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

	var remote net.IP
	if Net == "tcp" {
		remote = w.RemoteAddr().(*net.TCPAddr).IP
	} else {
		remote = w.RemoteAddr().(*net.UDPAddr).IP
	}
	logger.Info("%s lookup　%s", remote, Q.String())

	IPQuery := h.isIPQuery(q)

	// Query hosts
	if settings.Hosts.Enable && IPQuery > 0 {
		if ips, ok := h.hosts.Get(Q.qname, IPQuery); ok {
			m := new(dns.Msg)
			m.SetReply(req)

			switch IPQuery {
			case _IP4Query:
				rr_header := dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    settings.Hosts.TTL,
				}
				for _, ip := range ips {
					a := &dns.A{rr_header, ip}
					m.Answer = append(m.Answer, a)
				}
			case _IP6Query:
				rr_header := dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    settings.Hosts.TTL,
				}
				for _, ip := range ips {
					aaaa := &dns.AAAA{rr_header, ip}
					m.Answer = append(m.Answer, aaaa)
				}
			}

			w.WriteMsg(m)
			logger.Debug("%s found in hosts file", Q.qname)
			return
		} else {
			logger.Debug("%s didn't found in hosts file", Q.qname)
		}
	}

	writeMsgFromCache := func(m *dns.Msg) {
		// we need this copy against concurrent modification of Id
		msg := *m
		msg.Id = req.Id
		w.WriteMsg(&msg)
	}

	key := KeyGen(Q)
	mesg, err := h.cache.Get(key)
	if err != nil {
		if errExpired, ok := err.(KeyExpired); ok {
			logger.Debug("%s return old cache and refresh", Q.String())
			writeMsgFromCache(errExpired.Msg)
			// refresh cache now
			go h.refresh(key, Q, Net, req)
			return
		}
		if _, err = h.negCache.Get(key); err != nil {
			logger.Debug("%s didn't hit cache", Q.String())
		} else {
			logger.Debug("%s hit negative cache", Q.String())
			dns.HandleFailed(w, req)
			return
		}
	} else {
		logger.Debug("%s hit cache", Q.String())
		writeMsgFromCache(mesg)
		return
	}

	mesg, err = h.resolver.Lookup(Net, req)

	if err != nil {
		logger.Warn("Resolve query error %s", err)
		dns.HandleFailed(w, req)

		// cache the failure, too!
		if err = h.negCache.Set(key, nil); err != nil {
			logger.Warn("Set %s negative cache failed: %v", Q.String(), err)
		}
		return
	}

	w.WriteMsg(mesg)

	if len(mesg.Answer) > 0 {
		err = h.cache.Set(key, mesg)
		if err != nil {
			logger.Warn("Set %s cache failed: %s", Q.String(), err.Error())
		}
		logger.Debug("Insert %s into cache", Q.String())
	}
}

func (h *GODNSHandler) refresh(key string, Q Question, Net string, req *dns.Msg) {
	// use singleflight.Group to ensure only one query is executing for same key.
	h.refreshGroup.Do(key, func() (v interface{}, err error) {
		mesg, err := h.resolver.Lookup(Net, req)

		if err != nil {
			logger.Debug("Resolve query on refresh error %s", err)
			return
		}

		if len(mesg.Answer) > 0 {
			err = h.cache.Set(key, mesg)
			if err != nil {
				logger.Debug("Set %s cache failed on refresh: %s", Q.String(), err.Error())
			}
			logger.Debug("Insert %s into cache on refresh", Q.String())
		}
		return
	})
}

func (h *GODNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	h.do("tcp", w, req)
}

func (h *GODNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	h.do("udp", w, req)
}

func (h *GODNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
