.PHONY: help clean test test-ci package publish

LAMBDA_BUCKET ?= "pennsieve-cc-lambda-functions-use1"
WORKING_DIR   ?= "$(shell pwd)"
API_DIR ?= "api"
SERVICE_NAME  ?= "datasets-service"

SERVICE_EXEC  ?= "datasets_service"
SERVICE_PACK  ?= "datasetsService"
PACKAGE_NAME  ?= "${SERVICE_NAME}-${IMAGE_TAG}.zip"

MANIFEST_WORKER_NAME ?= "manifest-worker"
MANIFEST_WORKER_EXEC  ?= "manifest_worker"
MANIFEST_WORKER_SERVICE_PACK  ?= "manifestWorker"
MANIFEST_WORKER_PACKAGE_NAME ?= "${MANIFEST_WORKER_NAME}-${IMAGE_TAG}.zip"

.DEFAULT: help

help:
	@echo "Make Help for $(SERVICE_NAME)"
	@echo ""
	@echo "make clean			- spin down containers and remove db files"
	@echo "make test			- run dockerized tests locally"
	@echo "make test-ci			- run dockerized tests for Jenkins"
	@echo "make package			- create venv and package lambda function"
	@echo "make publish			- package and publish lambda function"

# Start the local versions of docker services
local-services:
	docker-compose -f docker-compose.test.yml down --remove-orphans
	docker-compose -f docker-compose.test.yml up -d pennsievedb

# Run tests locally
#test2: local-services
#	#./run-tests.sh localtest.env
#	docker-compose -f docker-compose.test.yml down --remove-orphans
#	make clean

test:
	docker-compose -f docker-compose.test.yml down --remove-orphans
	docker-compose -f docker-compose.test.yml up --exit-code-from local_tests local_tests


# Run test coverage locally
test-coverage: local-services
	./run-test-coverage.sh localtest.env
	docker-compose -f docker-compose.test.yml down --remove-orphans
	make clean

# Run dockerized tests (used on Jenkins)
test-ci:
	docker-compose -f docker-compose.test.yml down --remove-orphans
	@IMAGE_TAG=$(IMAGE_TAG) docker-compose -f docker-compose.test.yml up --exit-code-from=ci-tests ci-tests

clean: docker-clean
	rm -fR lambda/bin

# Spin down active docker containers.
docker-clean:
	docker-compose -f docker-compose.test.yml down

# Build lambda and create ZIP file
package:
	@echo ""
	@echo "***********************"
	@echo "*   Building lambda   *"
	@echo "***********************"
	@echo ""
	cd lambda/service; \
  		env GOOS=linux GOARCH=amd64 go build -o $(WORKING_DIR)/lambda/bin/$(SERVICE_PACK)/$(SERVICE_EXEC); \
		cd $(WORKING_DIR)/lambda/bin/$(SERVICE_PACK)/ ; \
			zip -r $(WORKING_DIR)/lambda/bin/$(SERVICE_PACK)/$(PACKAGE_NAME) .
	@echo ""
	@echo "***************************************"
	@echo "*   Building manifest worker lambda   *"
	@echo "***************************************"
	@echo ""
	cd lambda/manifestWorker; \
		env GOOS=linux GOARCH=amd64 go build -o $(WORKING_DIR)/lambda/bin/$(MANIFEST_WORKER_SERVICE_PACK)/$(MANIFEST_WORKER_EXEC); \
		cd $(WORKING_DIR)/lambda/bin/$(MANIFEST_WORKER_SERVICE_PACK)/ ; \
			zip -r $(WORKING_DIR)/lambda/bin/$(MANIFEST_WORKER_SERVICE_PACK)/$(MANIFEST_WORKER_PACKAGE_NAME) .

# Copy Service lambda to S3 location
publish:
	@make package
	@echo ""
	@echo "*************************"
	@echo "*   Publishing lambda   *"
	@echo "*************************"
	@echo ""
	aws s3 cp $(WORKING_DIR)/lambda/bin/$(SERVICE_PACK)/$(PACKAGE_NAME) s3://$(LAMBDA_BUCKET)/$(SERVICE_NAME)/
	rm -rf $(WORKING_DIR)/lambda/bin/$(SERVICE_PACK)/$(PACKAGE_NAME)
	@echo ""
	@echo "********************************"
	@echo "*   Publishing worker lambda   *"
	@echo "********************************"
	@echo ""
	aws s3 cp $(WORKING_DIR)/lambda/bin/$(MANIFEST_WORKER_SERVICE_PACK)/$(MANIFEST_WORKER_PACKAGE_NAME) s3://$(LAMBDA_BUCKET)/$(MANIFEST_WORKER_NAME)/
	rm -rf $(WORKING_DIR)/lambda/bin/$(MANIFEST_WORKER_SERVICE_PACK)/$(MANIFEST_WORKER_PACKAGE_NAME)

# Run go mod tidy on modules
tidy:
	cd ${WORKING_DIR}/lambda/service; go mod tidy
	cd ${WORKING_DIR}/api; go mod tidy

