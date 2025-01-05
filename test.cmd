file_content=$(cat test.txt.b64)
curl -X POST -H "Content-Type: application/json" -d '{"FileBoby": "'"$file_content"'", "FileName": "my_test_file.txt"}' http://localhost:8080/SaveFileToStorage