package traefikdisolver

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/kyaxcorp/traefikdisolver/providers"
	"github.com/kyaxcorp/traefikdisolver/providers/cloudflare"
	"github.com/kyaxcorp/traefikdisolver/providers/cloudfront"
)

func (r *Disolver) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	trustResult := r.trust(req.RemoteAddr,req)
	if trustResult.isFatal {
		http.Error(rw, "Unknown source", http.StatusInternalServerError)
		return
	}
	if trustResult.isError {
		http.Error(rw, "Unknown source", http.StatusBadRequest)
		return
	}
	if trustResult.directIP == "" {
		http.Error(rw, "Unknown source", http.StatusUnprocessableEntity)
		return
	}
	if trustResult.trusted {
		switch r.provider {
		case providers.Cloudflare:
			if req.Header.Get(cloudflare.CfVisitor) != "" {
				var cfVisitorValue CFVisitorHeader
				if err := json.Unmarshal([]byte(req.Header.Get(cloudflare.CfVisitor)), &cfVisitorValue); err != nil {
					req.Header.Set(cloudflare.XCfTrusted, "danger")
					req.Header.Del(cloudflare.CfVisitor)
					req.Header.Del(cloudflare.ClientIPHeaderName)
					r.next.ServeHTTP(rw, req)
					return
				}
				req.Header.Set(xForwardProto, cfVisitorValue.Scheme)
			}
		case providers.Cloudfront:
		}

		switch r.provider {
		case providers.Cloudflare:
			req.Header.Set(cloudflare.XCfTrusted, "yes")
		case providers.Cloudfront:
		}

		var clientIPHeaderName string
		var clientIP string
		switch r.provider {
		case providers.Auto:
			if req.Header.Get(cloudflare.ClientIPHeaderName) != "" {
				clientIPHeaderName = cloudflare.ClientIPHeaderName
			} else if req.Header.Get(cloudfront.ClientIPHeaderName) != "" {
				clientIPHeaderName = cloudfront.ClientIPHeaderName
			}

			// TODO: check if trusted...? or the ones that have been added by the user are combined
		default:
			clientIPHeaderName = r.clientIPHeaderName
		}

		if clientIPHeaderName != "" {
			rawIP := req.Header.Get(clientIPHeaderName)
			var err error
			clientIP, _, err = net.SplitHostPort(rawIP)
			if err != nil {
				clientIP = rawIP
			}
		}

		if clientIP != "" {
			req.Header.Set(xForwardFor, clientIP)
			req.Header.Set(xRealIP, clientIP)
		}
	} else {
		switch r.provider {
		case providers.Cloudflare:
			req.Header.Set(cloudflare.XCfTrusted, "no")
			req.Header.Del(cloudflare.CfVisitor)
			req.Header.Del(cloudflare.ClientIPHeaderName)
		case providers.Cloudfront:
		}
		req.Header.Set(xRealIP, trustResult.directIP)
	}
	r.next.ServeHTTP(rw, req)
}
