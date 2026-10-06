// Code generated from Swagger annotations. DO NOT EDIT.
package docs

import (
	_ "embed"
	"github.com/swaggo/swag"
)

//go:embed swagger.json
var swaggerJSON string

// SwaggerInfo holds the API metadata and specification.
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	BasePath:         "/",
	Title:            "Lecture Order Demo API",
	Description:      "In-memory order API for the lecture From Classroom Code to Production Software.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  swaggerJSON,
}

func init() { swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo) }
