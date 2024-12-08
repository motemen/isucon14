package main

import (
	"bytes"
	"context"
	"github.com/goccy/go-json"
	"errors"
	"net/http"
	"time"
)

var erroredUpstream = errors.New("errored upstream")

type paymentGatewayPostPaymentRequest struct {
	Amount int `json:"amount"`
}

type paymentGatewayGetPaymentsResponseOne struct {
	Amount int    `json:"amount"`
	Status string `json:"status"`
}

var (
	errBadRequest        = errors.New("bad request")
	errKeyExpired        = errors.New("key is expired or something")
	errUnexpected        = errors.New("unexpected status code")
	errPaymentProcessing = errors.New("payment processing")
)

func requestPaymentGatewayPostPayment(ctx context.Context, paymentGatewayURL string, token string, param *paymentGatewayPostPaymentRequest, rideID string) error {
	b, err := json.Marshal(param)
	if err != nil {
		return err
	}

	// see webapp/payment_mock/openapi.yaml for spec
	retry := 0
	for {
		err := func() error {
			var req *http.Request
			if retry < 1 {
				var err error
				req, err = http.NewRequestWithContext(ctx, http.MethodPost, paymentGatewayURL+"/payments", bytes.NewBuffer(b))
				if err != nil {
					return err
				}
			} else {
				var err error
				req, err = http.NewRequestWithContext(ctx, http.MethodGet, paymentGatewayURL+"/payments", nil)
				if err != nil {
					return err
				}
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Idempotency-Key", rideID)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			switch res.StatusCode {
			case http.StatusOK, http.StatusNoContent:
				return nil
			case 400:
				return errBadRequest
			case 422:
				return errKeyExpired
			case 409:
				return errPaymentProcessing
			default:
				return errUnexpected
			}
		}()
		if err != nil {
			if err == errPaymentProcessing && retry < 5 {
				retry++
				time.Sleep(100 * time.Millisecond)
				continue
			} else {
				return err
			}
		}
		break
	}

	return nil
}
