package main

import (
	"os"
	"whaleisle/internal/desktop/pet"
)

// Configure before any companion can be constructed. An explicit development
// source supports go-run's temporary executable without searching the checkout.
func init() { configurePetRenderAssets() }

func configurePetRenderAssets() string {
	return pet.ConfigureRenderAssets(developmentApp(), os.Getenv("WHALEISLE_PET_RENDER_ASSETS"))
}
