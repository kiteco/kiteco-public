OUTPUT := process-docs.tar

image: Dockerfile
	docker build -t kiteco/process-docs .

save: image
	docker save kiteco/process-docs > $(OUTPUT) 
