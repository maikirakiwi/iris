package menu

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/manifoldco/promptui"
	"github.com/stripe/stripe-go/v76"

	DB "iris/v2/database"
	"iris/v2/models"
	stripeapi "iris/v2/stripe-api"
)

func ListInventory() []string {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return []string{}
	}
	readableInventory := []string{}
	for _, inv := range allInventory {
		readableInventory = append(readableInventory, fmt.Sprintf("%s (%d left)", inv.DisplayName, inv.Quantity))
	}
	return readableInventory

}

func AddTrackingInventory() {
	invEntry := &models.Inventory{}
	products, err := stripeapi.GetAllProduct(true)
	if err != nil {
		println("There are no active products on Stripe")
		return
	}

	readableFields := []string{}
	for _, product := range products {
		readableFields = append(readableFields, product.Name)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Track",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	invEntry.DisplayName = products[index].Name
	invEntry.Product = products[index].ID

	prompt := promptui.Prompt{
		Label: "Enter current quantity in stock",
	}
	initialInventory, err := prompt.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	intInventory, err := strconv.Atoi(initialInventory)
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	invEntry.Quantity = int64(intInventory)

	prompt = promptui.Prompt{
		Label: "Track inventory on existing Session Links that have this item? (y/n)",
	}
	cascadeTrack, _ := prompt.Run()
	if cascadeTrack == "y" {
		allLinks := []models.SessionLink{}
		err := DB.Conn.Find(&allLinks).Error
		if err != nil {
			println("Error: %v\n", err)
			return
		}
		for _, link := range allLinks {
			prodInLink := stripeapi.GetProductsInLink(link.LinkID)
			if slices.Contains(prodInLink, products[index].ID) {
				// Too lazy to test associations/many2many
				invEntry.SessionLinkIDs = append(invEntry.SessionLinkIDs, link.LinkID)
				link.TrackingInventoryIDs = append(link.TrackingInventoryIDs, invEntry.Product)
				DB.Conn.Save(&link)
				println("Now tracking " + products[index].Name + " on link " + link.Nickname)
			}
		}
	}

	DB.Conn.Create(&invEntry)
}

func removeProdFromLinks(inv []string, product string) []string {
	newInv := []string{}
	for _, entry := range inv {
		if entry != product {
			newInv = append(newInv, entry)
		}
	}
	return newInv
}

func RemoveTrackingInventory() {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	readableFields := []string{}
	for _, inv := range allInventory {
		readableFields = append(readableFields, inv.DisplayName)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Stop Tracking",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}

	for _, linkID := range allInventory[index].SessionLinkIDs {
		link := models.SessionLink{}
		DB.Conn.Where(&models.SessionLink{LinkID: linkID}).First(&link)
		link.TrackingInventoryIDs = removeProdFromLinks(link.TrackingInventoryIDs, allInventory[index].Product)
		DB.Conn.Save(&link)
	}

	DB.Conn.Delete(&allInventory[index])
}

func ChangeInventoryQuantity() {
	allInventory := []models.Inventory{}
	err := DB.Conn.Find(&allInventory).Error
	if err != nil {
		println("Error: %v\n", err)
		return
	}
	readableFields := []string{}
	for _, inv := range allInventory {
		readableFields = append(readableFields, inv.DisplayName)
	}
	promptSelect := promptui.Select{
		Label: "Select Product to Change Quantity",
		Items: readableFields,
	}
	index, _, err := promptSelect.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	prompt := promptui.Prompt{
		Label: "Enter new quantity",
	}
	newQuantity, err := prompt.Run()
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	intQuantity, err := strconv.Atoi(newQuantity)
	if err != nil {
		println("Invalid Input: %v\n", err.Error())
		return
	}
	allInventory[index].Quantity = int64(intQuantity)
	DB.Conn.Save(&allInventory[index])
}

// Used by webhook server
func UpdateInventory(product string, decrementQuantity int64) {
	inventory := models.Inventory{}
	err := DB.Conn.Where(&models.Inventory{Product: product}).First(&inventory).Error
	if err != nil {
		slog.Error("Error while searching database for product " + product + err.Error())
		return
	}
	inventory.Quantity = inventory.Quantity - decrementQuantity
	if inventory.Quantity > 0 {
		ClampAdjustableQty(inventory.SessionLinkIDs, product, inventory.Quantity)
	} else {
		RemoveItemFromLink(inventory.SessionLinkIDs, product)
	}

	DB.Conn.Save(&inventory)
}

func ClampAdjustableQty(links []string, product string, qty int64) {
	for _, link := range links {
		seslink := &models.SessionLink{}
		err := DB.Conn.Where(&models.SessionLink{LinkID: link}).First(seslink).Error
		if err != nil {
			slog.Error("Error while getting link from DB: " + err.Error())
			return
		}
		for _, li := range seslink.Params.LineItems {
			price := &models.Price{}
			err := DB.Conn.Where(&models.Price{PriceID: *li.Price}).First(price).Error
			if err != nil {
				slog.Error("Error while getting price from DB: " + err.Error())
				return
			}
			if price.Product == product {
				if li.AdjustableQuantity != nil {
					if *li.AdjustableQuantity.Maximum > qty {
						li.AdjustableQuantity.Maximum = stripe.Int64(qty)
					}
					if *li.AdjustableQuantity.Minimum == *li.AdjustableQuantity.Maximum {
						li.AdjustableQuantity = nil
					}
				}

				if *li.Quantity > qty {
					li.Quantity = stripe.Int64(qty)
				}
				break
			}

		}

		DB.Conn.Save(&seslink)
	}
}

func RemoveItemFromLink(links []string, product string) {
	for _, link := range links {
		seslink := &models.SessionLink{}
		err := DB.Conn.Where(&models.SessionLink{LinkID: link}).First(seslink).Error
		if err != nil {
			slog.Error("Error while getting link from DB: " + err.Error())
			return
		}
		if len(seslink.Params.LineItems) == 1 {
			seslink.Active = false
			DB.Conn.Save(&seslink)
			continue
		}
		newLineItems := []*stripe.CheckoutSessionLineItemParams{}
		for _, li := range seslink.Params.LineItems {
			price := &models.Price{}
			err := DB.Conn.Where(&models.Price{PriceID: *li.Price}).First(price).Error
			if err != nil {
				slog.Error("Error while getting price from DB: " + err.Error())
				return
			}
			if price.Product != product {
				newLineItems = append(newLineItems, li)
			}

		}
		seslink.Params.LineItems = newLineItems
		DB.Conn.Save(&seslink)
	}
}
