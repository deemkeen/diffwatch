#!/bin/bash
# Test rapid file updates

echo "Initial content" > test_data/test.txt

# Simulate rapid updates
for i in {1..20}; do
    echo "Update $i at $(date +%H:%M:%S.%N)" >> test_data/test.txt
    sleep 0.1
done

# Test lock file behavior (should be filtered)
for i in {1..10}; do
    touch test_data/test.LOCK
    rm -f test_data/test.LOCK
    sleep 0.05
done

echo "Test complete"
