#!/bin/bash
set -xe

# Customize trace output for commands to stand out
PS4='\[\e[36m\]RAN COMMAND: \[\e[m\]'

MGC_CLI=${MGC_CLI:-./mgc}

# 1. Define test variables
test_directory="test_directory"
test_bucket="test-bucket"
test_bucket_copy="test-bucket-copy"
download_test_directory="download_result"

# 2. Create test directory and files
mkdir -p $test_directory

for i in {1..5}
do
    echo "Test file $i" > "./$test_directory/test_file_$i.txt"
done

# 3. Create Bucket
$MGC_CLI object-storage buckets create $test_bucket

# 4. Upload file to bucket
$MGC_CLI object-storage objects upload ./$test_directory/test_file_1.txt $test_bucket/test_file_1.txt

# 5. Download file from bucket
$MGC_CLI object-storage objects download $test_bucket/test_file_1.txt ./$download_test_directory/test_file_1.txt

# 6. Compare the uploaded file and the downloaded file
if ! diff ./$test_directory/test_file_1.txt ./$download_test_directory/test_file_1.txt > /dev/null; then
    echo "Error: uploaded file and downloaded file content mismatch for test_file_1.txt (download)"
fi

rm ./$download_test_directory/test_file_1.txt

# 7. Delete file from bucket
$MGC_CLI object-storage objects delete $test_bucket/test_file_1.txt

# 8. Upload test directory to bucket
$MGC_CLI object-storage objects upload-dir ./$test_directory $test_bucket

# 9. Download test directory
$MGC_CLI object-storage objects download-all $test_bucket ./$download_test_directory

# 10. Compare the uploaded files and the downloaded files
for i in {1..5}; do
    if ! diff ./$test_directory/test_file_$i.txt ./$download_test_directory/$test_directory/test_file_$i.txt > /dev/null; then
        echo "Error: uploaded file and downloaded file content mismatch for test_file_$i.txt (download-all)"
    fi
done

# 11. Create bucket to copy file to
$MGC_CLI object-storage buckets create $test_bucket_copy

# 12. Upload file to be copied to first bucket
$MGC_CLI object-storage objects upload ./$test_directory/test_file_1.txt $test_bucket/test_file_1.txt

# 13. Copy file to new bucket
$MGC_CLI object-storage objects copy $test_bucket/test_file_1.txt $test_bucket_copy/test_file_1_copy.txt

# 14. Download copy file
$MGC_CLI object-storage objects download $test_bucket_copy/test_file_1_copy.txt ./$download_test_directory/test_file_1_copy.txt

# 15. Compare the original file and the copy file
if ! diff ./$test_directory/test_file_1.txt ./$download_test_directory/test_file_1_copy.txt > /dev/null; then
    echo "Error: original file and copy file content mismatch for test_file_1.txt (copy)"
fi

# 16. Delete all files from bucket
$MGC_CLI object-storage objects delete-all $test_bucket

# 17. Delete buckets
$MGC_CLI object-storage buckets delete $test_bucket
$MGC_CLI object-storage buckets delete $test_bucket_copy

# 18. Delete test directory and files
rm -r ./$test_directory
rm -r ./$download_test_directory
