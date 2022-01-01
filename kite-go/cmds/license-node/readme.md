## Testing with Stripe CLI

#### 1. Configure Strip CLI

Do step 1 and 2 from the page https://stripe.com/docs/stripe-cli for a quick intro on how to setup stripe CLI 

#### 2. Listen events with Stripe CLI

`stripe listen --forward-to localhost:9090/api/account/stripe-webhook`
That connects Stripe to the stripe test server instance. It prints a line with the webhook signing secret.

#### 3. Run Stripe Test 
Execute the `license-node` to use the signing key of the stripe-cli.

```bash
export STRIPE_WEBHOOK_SECRET="<signing key printed by stripe-cli>"
go build .
./license-node
```

#### Trigger event with the web interface
 Visit http://localhost:9090 and do actions, that should trigger webhook requests. You can use the card `4242 4242 4242 4242` to simulate payments. 
 
 