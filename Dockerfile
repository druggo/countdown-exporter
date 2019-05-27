FROM scratch

ENV COUNTDOWN_EXPTR_DEADLINES_FILE deadlines.yaml
ENV COUNTDOWN_EXPTR_DEADLINES_FILE_TYPE yaml
ENV COUNTDOWN_EXPTR_HTTP_PORT 9208
ENV COUNTDOWN_EXPTR_CHECK_INTERVAL_SECS 60

COPY countdown-exporter /countdown-exporter
COPY deadlines.yaml /deadlines.yaml

ENTRYPOINT ["/countdown-exporter"]
