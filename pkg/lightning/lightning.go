package lightning

import (
	"regexp"
	"strings"
)

// invoiceShape matches what a BOLT11 payment request looks like: "ln", a
// network prefix (bc mainnet, tb testnet, tbs signet, bcrt regtest, sb simnet),
// an optional amount with multiplier, the "1" separator and the data part, as
// one token. It deliberately does not verify the checksum or signature: a
// mistyped invoice should still reach the pay handlers, whose decoder tells
// the user what is wrong instead of the input being treated as something else.
var invoiceShape = regexp.MustCompile(`^ln(bcrt|bc|tbs|tb|sb)[0-9]*[munp]?1[0-9a-z]+$`)

// IsInvoice reports whether message looks like a single BOLT11 payment
// request, optionally prefixed with "lightning:". Case is ignored because QR
// codes usually encode invoices in upper case.
func IsInvoice(message string) bool {
	message = strings.TrimPrefix(strings.ToLower(message), "lightning:")
	return invoiceShape.MatchString(message)
}

func IsLnurl(message string) bool {
	message = strings.ToLower(message)
	if strings.HasPrefix(message, "lnurl") || strings.HasPrefix(message, "lightning:lnurl") {
		// string must be a single word
		if !strings.Contains(message, " ") {
			return true
		}
	}
	return false
}

// lightningAddress matches a LUD-16 Lightning address: a username of
// a-z0-9-_.+ and a dotted domain. Case is ignored because users type
// addresses by hand.
var lightningAddress = regexp.MustCompile(`(?i)^[a-z0-9._+-]+@[a-z0-9-]+(\.[a-z0-9-]+)+$`)

// IsLightningAddress reports whether address is a plain name@domain Lightning
// address. Unlike an e-mail parser it rejects display names ("Bob <b@x.com>"),
// quoted local parts and surrounding whitespace, so those inputs are not
// routed to the Lightning-address payment path.
func IsLightningAddress(address string) bool {
	return lightningAddress.MatchString(address)
}
