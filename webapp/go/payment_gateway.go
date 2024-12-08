package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
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

func requestPaymentGatewayPostPayment(ctx context.Context, paymentGatewayURL string, token string, param *paymentGatewayPostPaymentRequest) error {
	b, err := json.Marshal(param)
	if err != nil {
		return err
	}

	// see webapp/payment_mock/openapi.yaml for spec
	idemKey := ulid.Make().String()
	retry := 0
	for {
		err := func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, paymentGatewayURL+"/payments", bytes.NewBuffer(b))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Idempotency-Key", idemKey)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			switch res.StatusCode {
			case http.StatusNoContent:
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
