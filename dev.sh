#!/bin/bash
# UGREEN Go Dev Container - convenience script
# Usage: ./dev.sh [command]
#   no args  → enter container shell
#   exec CMD → run command in container
#   stop     → stop container
#   start    → start container
#   rm       → remove container

CONTAINER="ugreen-go-dev"

case "${1:-shell}" in
  shell)
    docker exec -it "$CONTAINER" /bin/bash
    ;;
  exec)
    shift
    docker exec "$CONTAINER" "$@"
    ;;
  stop)
    docker stop "$CONTAINER"
    ;;
  start)
    docker start "$CONTAINER"
    ;;
  rm)
    docker stop "$CONTAINER" 2>/dev/null
    docker rm "$CONTAINER"
    ;;
  *)
    echo "Usage: $0 [shell|exec CMD|stop|start|rm]"
    ;;
esac
