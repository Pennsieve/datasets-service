# datasets-service

A serverless AWS Lambda service that provides dataset management endpoints for the Pennsieve platform. This service handles dataset-related operations including trashcan management and dataset manifest generation.

## Service Overview

The datasets-service is a Go-based serverless application deployed as an AWS Lambda function. It connects to a PostgreSQL database through RDS Proxy and integrates with S3 for manifest storage and SNS for asynchronous processing.

## Endpoints

### `/datasets/trashcan`
**Method:** GET  
**Description:** Retrieves paginated list of deleted items from a dataset's trashcan  
**Authentication:** Requires `ViewFiles` permission  
**Query Parameters:**
- `dataset_id` (required): The dataset node ID
- `root_node_id` (optional): Filter by root node/folder ID
- `limit` (optional): Number of items per page (default: 10, max: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response:** Returns a paginated list of trashcan items including package ID, name, node ID, type, and deletion state.

### `/datasets/manifest`
**Method:** GET  
**Description:** Generates and retrieves a dataset manifest containing metadata about all files in the dataset  
**Authentication:** Requires `ViewFiles` permission  
**Query Parameters:**
- `dataset_id` (required): The dataset node ID

**Response:** Returns a manifest with dataset metadata and file information including:
- Dataset details (name, description, license, tags, contributors)
- File paths and metadata (node IDs, file names, sizes, checksums)
- Manifest is stored in S3 with a presigned URL for download

## Architecture

- **Runtime:** Go with AWS Lambda (ARM64 architecture)
- **Database:** PostgreSQL via RDS Proxy
- **Storage:** S3 for manifest files
- **Infrastructure:** Terraform for IaC
- **VPC:** Deployed in private subnets with security group configuration
