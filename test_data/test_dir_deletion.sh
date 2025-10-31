#!/bin/bash
# Test directory deletion behavior

echo "Creating test directory with files..."
mkdir -p test_data/temp_dir
cat > test_data/temp_dir/file1.txt << 'CONTENT'
File 1 content
line 2
line 3
CONTENT

cat > test_data/temp_dir/file2.txt << 'CONTENT'
File 2 content
another line
CONTENT

echo "Waiting 2 seconds..."
sleep 2

echo "Modifying file1..."
echo "New content added" >> test_data/temp_dir/file1.txt

echo "Waiting 2 seconds..."
sleep 2

echo "Deleting entire directory..."
rm -rf test_data/temp_dir

echo "Test complete - you should see deletion diffs for the files"
sleep 2
