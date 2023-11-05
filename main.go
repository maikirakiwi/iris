package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/gorm"

	DB "iris/v2/database"
	"iris/v2/menu"
	"iris/v2/models"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		slog.Warn("Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	settings := DB.GetSettings()
	stripe.Key = settings.ApiKey

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), settings.WebhookEndpointSecret)
	if err != nil {
		slog.Error("Error verifying webhook signature: " + err.Error())
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}

	// If error raised while handling event, return bad request.
	if !checkoutHandler(event) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func checkoutHandler(event stripe.Event) bool {
	// Handle the checkout.session.completed event
	if event.Type == "checkout.session.completed" {
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			slog.Error("Error parsing webhook JSON: " + err.Error())
			return false
		}

		if session.PaymentStatus == "paid" {
			link := &models.PaymentLink{}
			db_res := DB.Conn.Where(&models.PaymentLink{LinkID: session.PaymentLink.ID}).First(&link)
			if db_res.Error != nil {
				if errors.Is(db_res.Error, gorm.ErrRecordNotFound) {
					return true
				}
				slog.Error("Error while reading: " + db_res.Error.Error())
				return false
			}

			link.Used++
			if link.Used == link.MaxUses {
				link.Active = false
				_, err := paymentlink.Update(
					link.LinkID,
					&stripe.PaymentLinkParams{
						Active: stripe.Bool(false),
					},
				)
				if err != nil {
					slog.Error("Error while changing link on Stripe: " + err.Error())
					return false
				}
			}
			slog.Info("Link " + link.LinkID + " now used " + fmt.Sprintf("%d", link.Used) + " times")
			db_res = DB.Conn.Save(&link)
			if db_res.Error != nil {
				slog.Error("Error while saving link paid event to database: " + db_res.Error.Error())
				return false
			}
		}
	}

	return true
}

func serveWeb() {
	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Starting server on port 4242")
	http.ListenAndServe(":4242", nil)
}

func main() {
	if _, err := os.Stat("./data.db"); err != nil {
		DB.Init()
		menu.ChangeApiKey()
		menu.ChangeEndpointSecret()
	} else {
		DB.Init()
	}

	if len(os.Args) > 1 && os.Args[1] == "menu" {
		menu.Entry()
	} else {
		serveWeb()
	}

}
