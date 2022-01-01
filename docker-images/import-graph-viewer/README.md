To run this docker image you must supply AWS credentials:

```
docker run -i -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -P XXXXXXX/import-graph-viewer
```
