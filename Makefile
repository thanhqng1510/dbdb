BINARY_NAME=dbdb
OUTPUT_DIR=./bin

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@rm -f $(OUTPUT_DIR)/$(BINARY_NAME)

install-tools:
	@mkdir -p $(OUTPUT_DIR)
	@echo "Installing Air (live-reloading tool)..."
	GOBIN=$(CURDIR)/$(OUTPUT_DIR) go install github.com/air-verse/air

air:
	@echo "Starting $(BINARY_NAME) with Air for live reloading"
	$(OUTPUT_DIR)/air

.PHONY: build clean install-tools air
