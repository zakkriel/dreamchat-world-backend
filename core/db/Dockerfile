# Pinned postgres:16 family — the single image shared by local dev, CI, and the deployed (Railway) family.
# The DoD determinism check is only authoritative against THIS environment.
FROM postgres:16
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      postgresql-16-pgtap \
      libtap-parser-sourcehandler-pgtap-perl \
 && rm -rf /var/lib/apt/lists/*
# postgresql-16-pgtap = pgTAP extension SQL; libtap-parser-sourcehandler-pgtap-perl = /usr/bin/pg_prove.
