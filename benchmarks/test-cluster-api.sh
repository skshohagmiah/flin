#!/bin/bash

echo "Testing /cluster API endpoint..."
echo ""

# Wait for cluster to be ready
sleep 2

echo "Querying http://localhost:7080/cluster..."
curl -s http://localhost:7080/cluster | python3 -m json.tool

echo ""
echo "Done!"
