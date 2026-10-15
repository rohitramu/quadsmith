#!/bin/sh

EXTRA_ARGS=""
if [ "$UPDATE_SNAPSHOTS" = "true" ]; then
  EXTRA_ARGS="--update-snapshots"
fi

npx playwright test $EXTRA_ARGS "$@"
EXIT_CODE=$?

if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ]; then
  chown -R "$HOST_UID:$HOST_GID" /app/snapshots /app/playwright-report /app/test-results 2>/dev/null || true
fi

exit $EXIT_CODE
