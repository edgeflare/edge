package handler

import (
	"net"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var validate = validator.New()

func init() {
	// Register custom validation for hostname
	err := validate.RegisterValidation("hostname", isHostname)
	if err != nil {
		zap.L().Fatal("Failed to register 'hostname' validation", zap.Error(err))
	}

	// Register custom validation for IP address
	err = validate.RegisterValidation("ipaddress", isIPAddress)
	if err != nil {
		zap.L().Fatal("Failed to register 'ipaddress' validation", zap.Error(err))
	}
}

func isHostname(fl validator.FieldLevel) bool {
	hostname := fl.Field().String()
	_, err := net.LookupHost(hostname)
	return err == nil
}

func isIPAddress(fl validator.FieldLevel) bool {
	ip := fl.Field().String()
	return net.ParseIP(ip) != nil
}
