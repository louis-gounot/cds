define MINIO_CONFIG
export AWS_DEFAULT_REGION=us-east-1
export S3_BUCKET=test-it
export MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
export MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_ENDPOINT_URL=http://$$(hostname):9000
endef

MINIO_CONTAINER_ID = $(shell docker ps -f name=minio1 -q)

clean:
	@rm -f $(MINIO_CONTAINER_ID) $(MINIO_RC)
	@docker kill minio1 || true && docker rm minio1 || true

minio_start: $(MINIO_RC)
	@if [ -z "$(MINIO_CONTAINER_ID)" ]; then \
		docker rm minio1 >/dev/null 2>&1 || true; \
		echo "starting minio container"; \
		docker run -d -p 9000:9000 --name minio1 \
  			-e "MINIO_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE" \
  			-e "MINIO_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" minio/minio server /data; \
	fi; \
	$(info # Here are the minio configuration variables)
	$(info $(MINIO_CONFIG))

minio_reset_bucket:
	@docker run --rm --link minio1:minio --entrypoint sh minio/mc -c "\
		while ! nc -z minio 9000; do echo 'Wait minio to startup...' && sleep 0.1; done; \
		sleep 5 && \
		mc config host add myminio http://minio:9000 AKIAIOSFODNN7EXAMPLE wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY && \
		mc rm -r --force myminio/test-it  || true && \
		mc mb myminio/test-it || true \
		"