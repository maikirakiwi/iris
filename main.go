package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentlink"
	"github.com/stripe/stripe-go/v76/webhook"
	json "github.com/sugawarayuuta/sonnet" // Faster and correct, drop in json parser
	"gorm.io/gorm"

	DB "stripe-handler/v2/database"
	"stripe-handler/v2/menu"
	"stripe-handler/v2/models"
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
		fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}

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
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			return false
		}

		if session.PaymentStatus == "paid" {
			link := &models.PaymentLink{}
			db_res := DB.Conn.Where(&models.PaymentLink{LinkID: session.PaymentLink.ID}).First(&link)
			if db_res.Error != nil {
				if errors.Is(db_res.Error, gorm.ErrRecordNotFound) {
					return true
				}
				println("Error: %v\n", db_res.Error)
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
					println("Error while changing link on Stripe: " + err.Error())
					return false
				}
			}
			println("Link " + link.LinkID + " now used " + fmt.Sprintf("%d", link.Used) + " times")
			db_res = DB.Conn.Save(&link)
			if db_res.Error != nil {
				println("Error: %v\n", db_res.Error)
				return false
			}
		}
	}

	return true
}

func serveWeb() {
	http.HandleFunc("/webhook", webhookHandler)
	println("Starting server on port 4242")
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
		menu.Main()
	} else {
		serveWeb()
	}

}
