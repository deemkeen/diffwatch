#!/bin/bash
# Test file deletion behavior

echo "Creating test file..."
cat > test_data/delete_me.txt << 'CONTENT'
This is a test file
with multiple lines
that will be deleted
to test deletion diff
CONTENT

echo "Waiting 2 seconds..."
sleep 2

echo "Modifying the file..."
echo "Added a new line" >> test_data/delete_me.txt

echo "Waiting 2 seconds..."
sleep 2

echo "Deleting the file..."
rm test_data/delete_me.txt

echo "Test complete - you should see the deletion diff"
sleep 2
