package httpx

import (
	"context"
	"net/http"
)

// Endpoint composes the pipeline every JSON endpoint follows:
//
//	decode (untrusted Req) → parse (validated In) → run (domain Out) → present (wire view)
//
// parse also sees the request, for boundary facts like the client IP. Only
// parse and present cross the boundary; run sees validated values only.
func Endpoint[Req, In, Out, View any](
	status int,
	parse func(*http.Request, Req) (In, error),
	run func(context.Context, In) (Out, error),
	present func(Out) View,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := DecodeJSON[Req](w, r)
		if err != nil {
			Error(w, err)
			return
		}
		in, err := parse(r, req)
		if err != nil {
			Error(w, err)
			return
		}
		out, err := run(r.Context(), in)
		if err != nil {
			Error(w, err)
			return
		}
		JSON(w, status, present(out))
	}
}
