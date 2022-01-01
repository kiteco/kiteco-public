package stripe

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/octobat"
)

var (
	octobatSecretKey string
	octobatPublicKey string
	beanieConfigID   string
)

// InitOctobat must be called before using octobat beanie, it allows to initialize correctly the Octobat API
func InitOctobat(publicKey, secretKey, configID string) {
	octobatSecretKey = secretKey
	octobatPublicKey = publicKey
	beanieConfigID = configID
}

// CreateSubscriptionCheckoutOctobat generate the required payload to send to octobat beanie (checkout system)
// for the user to subscribe to one of KitePro plans
func CreateSubscriptionCheckoutOctobat(name, email, userID, userIP, planID string, domain, cancelPage string) ([]byte, error) {
	if err := checkInit(); err != nil {
		return nil, err
	}

	info := octobat.BeanieServerlessSession{
		ClientReferenceID: userID,
		ConfigurationID:   beanieConfigID,
		SuccessURL:        fmt.Sprintf("https://%s/web/account/checkout/octobat-success", domain),
		CancelURL:         cancelPage,
		Items: []octobat.BeanieItem{{
			StripePlanID: planID,
			Quantity:     1,
		}},
		PrefillData: octobat.BeaniePrefillData{
			CustomerName:  name,
			CustomerEmail: email,
		},
		Metadata: octobat.BeanieKiteMetadata{
			KiteUserID: userID,
			IPAddress:  userIP,
		},
	}

	payload, err := json.Marshal(info)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while marshalling json payload for octobat checkout")
	}
	result := fmt.Sprintf(checkoutTemplate, octobatPublicKey, payload)
	return []byte(result), nil
}

// CreateSubscriptionCheckoutOctobatWithTrial generate the required payload to send to octobat beanie (checkout system)
// for the user to subscribe to one of KitePro plans.
// It uses Octobat's client/server model to setup a new session with a trial.
func CreateSubscriptionCheckoutOctobatWithTrial(trialDuration time.Duration, name, email, userID, userIP, planID string, domain, cancelPage string) ([]byte, error) {
	if err := checkInit(); err != nil {
		return nil, err
	}

	info := octobat.BeanieServerSession{
		Mode:              octobat.ModePayment,
		ClientReferenceID: userID,
		ConfigurationID:   beanieConfigID,
		SuccessURL:        fmt.Sprintf("https://%s/web/account/checkout/octobat-success", domain),
		CancelURL:         cancelPage,
		SubscriptionData: &octobat.SubscriptionData{
			TrialEnd: time.Now().Add(trialDuration).Unix(),
			Items: []octobat.BeanieItem{{
				StripePlanID: planID,
				Quantity:     1,
			}},
		},
		PrefillData: &octobat.BeaniePrefillData{
			CustomerName:  name,
			CustomerEmail: email,
		},
		Metadata: &octobat.BeanieKiteMetadata{
			KiteUserID:        userID,
			IPAddress:         userIP,
			KiteTrialDuration: int64(trialDuration.Round(time.Second).Seconds()),
		},
	}

	beanieClient := octobat.NewBeanieClient(octobatSecretKey)
	sessionResp, err := beanieClient.CreateSession(info)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating a new octobat session")
	}

	payload, err := json.Marshal(struct {
		SessionID string `json:"sessionId"`
	}{
		SessionID: sessionResp.ID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Error while marshalling JSON")
	}
	result := fmt.Sprintf(checkoutTemplate, octobatPublicKey, payload)
	return []byte(result), nil
}

const checkoutTemplate = `
<html>
<head>
  <script src="https://cdn.jsdelivr.net/gh/0ctobat/octobat-beanie.js@latest/dist/octobat-beanie.min.js"></script>
  <title>Redirection to Checkout</title>
</head>

<body>

<p id="error-message">
</p>

<script>
var beanie = OctobatBeanie('%s');
var payload = JSON.parse('%s');
beanie.redirectToBeanie(payload)
  .then(function (result) {
    if (result.error) {
      var displayError = document.getElementById('error-message');
      displayError.textContent = result.error.message;
    }
  }).catch(function (error) {
    console.log(error);
    var displayError = document.getElementById('error-message');
    displayError.textContent = "The payment system seems unreachable. Please try again later.";
  });
</script>
</body>
</html>
`
