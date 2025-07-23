package common

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ParseCidrs parses and validates CIDRs
func ParseCidrs(cidrs []string) (validCidrs []*net.IPNet, warnings []string, err error) {
	var parsed []*net.IPNet
	for _, cidr := range cidrs {
		if !strings.Contains(cidr, "/") {
			cidr = cidr + "/32"
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("provided cidr[%s] is invalid, it was skipped", cidr))
		}
		parsed = append(parsed, network)
	}
	return parsed, warnings, nil
}

// extractRequestIp extracts IP from X-Forwarded-For or RemoteAddr
func extractRequestIp(r *http.Request) (net.IP, error) {
	forwardedForHeader := r.Header.Get("X-Forwarded-For")
	if forwardedForHeader != "" {
		parts := strings.Split(forwardedForHeader, ",")
		if len(parts) > 0 {
			remoteIp := strings.TrimSpace(parts[0])
			parsed := net.ParseIP(remoteIp)
			if parsed != nil {
				return parsed, nil
			}
		}
	}
	remoteIp, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}
	parsed := net.ParseIP(remoteIp)
	if parsed == nil {
		return nil, errors.New("invalid remote ip")
	}
	return parsed, nil
}

// isIpAllowed checks if the IP is inside any of the allowed CIDRs
func isIpAllowed(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
