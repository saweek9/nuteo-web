// Package handlers contains the HTTP handlers — one file per concern.
//
// Handlers stay thin: parse input, call services, render templates.
package handlers

import (
	"github.com/nuteo/nuteo-web/internal/config"
	"github.com/nuteo/nuteo-web/internal/email"
	"github.com/nuteo/nuteo-web/internal/i18n"
	"github.com/nuteo/nuteo-web/internal/storage"
)

// Deps bundles everything handlers need.
type Deps struct {
	Cfg   *config.Config
	Store *storage.Store
	Mail  email.Sender
	I18n  *i18n.Bundle
}
