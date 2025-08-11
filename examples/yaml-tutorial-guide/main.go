// YAML Configuration Tutorial for weft
// ======================================
//
// This example demonstrates how to create your own YAML-based configuration
// system using the weft toolkit. You'll learn how to:
//
// 1. Define a custom specification struct with YAML tags
// 2. Implement validation for your configuration
// 3. Load YAML files using the generic config loader
// 4. Use your configuration with the weft engine
// 5. Generate code from templates based on your YAML config
//
// This tutorial uses a simple "database schema generator" as an example,
// but you can adapt this pattern for any type of code generation task.

package main

import (
	"embed"
	"flag"
	"fmt"
	"log"

	"github.com/cpcf/weft/config"
	"github.com/cpcf/weft/engine"
	"github.com/cpcf/weft/examples/yaml-tutorial-guide/spec"
	"github.com/cpcf/weft/processors"
)

// STEP 1: Embed your template files
// =================================
// The //go:embed directive allows you to embed your template files directly
// into your binary. This makes distribution easier and ensures templates
// are always available.

//go:embed templates
var templateFS embed.FS

func main() {
	// STEP 2: Set up command line flags
	// ================================
	// Provide a way for users to specify their YAML configuration file.
	// Always provide a fallback option (hardcoded config) for testing.

	configPath := flag.String("config", "", "Path to YAML configuration file")
	flag.Parse()

	// STEP 3: Load your configuration
	// ==============================
	// This is where the magic happens! Use the generic config.LoadYAML
	// function to load any YAML file into your custom struct type.

	var dbSchema *spec.DatabaseSchema
	var err error

	if *configPath != "" {
		// Load from YAML file using the generic toolkit loader
		fmt.Printf("📂 Loading configuration from %s...\n", *configPath)

		dbSchema = &spec.DatabaseSchema{}
		err = config.LoadYAML(*configPath, dbSchema)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		fmt.Printf("Configuration loaded successfully!\n")
	} else {
		fmt.Printf("Use --config to load from YAML file\n")
		return
	}

	// STEP 4: Create and configure the weft engine
	// ===============================================
	// The engine handles template processing and file generation.
	// You can customize the output directory and failure behavior.

	fmt.Printf("Setting up weft engine...\n")

	eng := engine.New(
		engine.WithOutputRoot("./generated"),
		engine.WithFailureMode(engine.FailFast),
	)

	// STEP 5: Add post-processors (optional but recommended)
	// =====================================================
	// Post-processors clean up generated files and add helpful metadata.

	eng.AddPostProcessor(processors.NewGoImports())                       // Fix Go imports
	eng.AddPostProcessor(processors.NewTrimWhitespace())                  // Clean up whitespace
	eng.AddPostProcessor(processors.NewAddGeneratedHeader("weft", ".go")) // Add generated headers

	// STEP 6: Create template context and generate code
	// ================================================
	// The context provides the embedded filesystem and output configuration.
	// The template data can be any Go value - your YAML config becomes available
	// in templates as template variables.

	ctx := engine.NewContext(templateFS, "./generated", dbSchema.Package)

	fmt.Printf("Generating database code for schema '%s'...\n", dbSchema.Name)
	fmt.Printf("   Package: %s\n", dbSchema.Package)
	fmt.Printf("   Tables: %d\n", len(dbSchema.Tables))

	// Render all templates in the "templates" directory
	// Your YAML configuration will be available in templates as {{ .Schema }}
	if err := eng.RenderDir(ctx, "templates", map[string]any{
		"Schema": dbSchema,
	}); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	// STEP 7: Success! Provide helpful next steps
	// ==========================================

	fmt.Printf("\nCode generation completed successfully!\n")
	fmt.Printf("Generated files in ./generated/\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("   1. cd generated\n")
	fmt.Printf("   2. go mod init %s\n", dbSchema.Package)
	fmt.Printf("   3. go mod tidy\n")
	fmt.Printf("   4. go build\n")

	if *configPath == "" {
		fmt.Printf("\nTip: Try creating your own YAML config file and run:\n")
		fmt.Printf("   go run main.go --config your-schema.yaml\n")
	}
}
