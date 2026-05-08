package http

import (
	"context"
	"net/http"
	"time"

	"github.com/openbao/openbao/sdk/v2/helper/consts"
	"github.com/openbao/openbao/vault"
	"github.com/sethvargo/go-limiter/httplimit"
)

func wrapIndexForwardHandler(mux http.Handler, core *vault.Core, props *vault.HandlerProperties) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqHeader := r.Header.Get(consts.IndexRequestHeaderName)
		if reqHeader != "" {
			// When we have a header, we have two options:
			//
			// 1. If listener is not configured for use with a server-side
			//    wait, we push this back on the client for client retries.
			// 2. Otherwise, we maybe do server-side waits. When this wait
			//    is not sufficient, we'll forward the request to the active
			//    node for processing as we want a server initiated retry.
			if props.ListenerConfig == nil || props.ListenerConfig.MaxIndexWait == 0 {
				if core.CheckIndexAndMaybeRetry(ctx, reqHeader) {
					// Don't force clients to wait too long; retry-after takes
					// a number of seconds to wait. We don't know the actual
					// latency but adding a 1 second delay is the minimum we
					// can set and at worst we cause clients to retry again.
					//
					// http.StatusTooManyRequests is chosen because it was
					// already implemented by the quota system and so should
					// be mostly invisible to rate limit quota-respecting
					// clients.
					w.Header().Add(httplimit.HeaderRetryAfter, "1")
					respondError(w, http.StatusTooManyRequests, nil)
					return
				}

				// We reached the desired index state; handle it locally.
				mux.ServeHTTP(w, r)
				return
			}

			// Our listener has a maximum server-side wait value, letting
			// us enter a server-side retry loop.
			maxWait := props.ListenerConfig.MaxIndexWait

			// Create a new context chained from the request's, bound to
			// the maximum server side wait context. This lets us forward
			// the request sooner if we encounter timeouts.
			ctx, cancel := context.WithTimeout(r.Context(), maxWait+100*time.Millisecond)
			defer cancel()

			if core.AwaitIndexAndMaybeForward(ctx, maxWait, reqHeader) {
				// Index was not reached, so forward the request and assume
				// the active node is up to date.
				forwardRequest(core, w, r)
				return
			}

			// Fall through to potentially handling the request locally.
		}

		mux.ServeHTTP(w, r)
		return
	})
}
