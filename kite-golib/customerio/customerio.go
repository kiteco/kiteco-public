package customerio

import (
	"os"

	customerio "github.com/customerio/go-customerio"
)

var cio *customerio.CustomerIO

func init() {
	// These environment variables will be set on the deployed machines.
	// Without this token, reporting is a NOOP, which is default behavior
	// (e.g while debugging/developing in a VM).
	siteID := os.Getenv("CUSTOMER_IO_SITE_ID")
	apiKey := os.Getenv("CUSTOMER_IO_API_KEY")
	if siteID == "" || apiKey == "" {
		return
	}

	cio = customerio.NewCustomerIO(siteID, apiKey)
	cio.SSL = true
}

// Track a single event on customer.io for the supplied user,
// if the CUSTOMERIO_SITE_ID and CUSTOMERIO_API_KEY are not set then
// this is a NOOP.
func Track(customerID string, eventName string, data map[string]interface{}) error {
	if cio == nil {
		return nil
	}

	err := cio.Track(customerID, eventName, data)
	return err
}

// Identify a single customer with customer.io and set their attributes,
// if the CUSTOMERIO_SITE_ID and CUSTOMERIO_API_KEY are not set then
// this is a NOOP.
func Identify(customerID string, attributes map[string]interface{}) error {
	if cio == nil {
		return nil
	}
	err := cio.Identify(customerID, attributes)
	return err
}
