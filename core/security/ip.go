package security

import (
	"fmt"
	"net"

	"github.com/labstack/echo/v4"
)

// IPExtractor returns the extractor Echo uses for RealIP. With no
// trustedProxies it reads the TCP peer only; otherwise X-Forwarded-For is
// honoured through peers inside the given CIDRs. It errors on a CIDR it
// cannot parse or on a catch-all range, which would let every client pick
// its own address.
func IPExtractor(trustedProxies []string) (echo.IPExtractor, error) {
	if len(trustedProxies) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	opts := []echo.TrustOption{echo.TrustLoopback(false), echo.TrustLinkLocal(false), echo.TrustPrivateNet(false)}
	for _, cidr := range trustedProxies {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parsing trusted proxy %q: %w", cidr, err)
		}
		if ones, _ := ipNet.Mask.Size(); ones == 0 {
			return nil, fmt.Errorf("trusted proxy %q matches every address", cidr)
		}
		opts = append(opts, echo.TrustIPRange(ipNet))
	}
	return echo.ExtractIPFromXFFHeader(opts...), nil
}
