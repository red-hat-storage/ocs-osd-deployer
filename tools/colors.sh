#!/bin/bash

BLACK='\033[0;30m'  # Black
RED='\033[0;31m'    # Red
GREEN='\033[0;32m'  # Green
YELLOW='\033[0;33m' # Yellow
BLUE='\033[0;34m'   # Blue
PURPLE='\033[0;35m' # Purple
CYAN='\033[0;36m'   # Cyan
WHITE='\033[0;37m'  # White
NC='\033[0m'        # No Color

function black {
    _colored_echo "${1}" ${BLACK}
}

function red {
    _colored_echo "${1}" ${RED}
}

function green {
    _colored_echo "${1}" ${GREEN}
}

function yellow {
    _colored_echo "${1}" ${YELLOW}
}

function blue {
    _colored_echo "${1}" ${BLUE}
}

function purple {
    _colored_echo "${1}" ${PURPLE}
}

function cyan {
    _colored_echo "${1}" ${CYAN}
}

function white {
    _colored_echo "${1}" ${WHITE}
}

function _colored_echo {
    echo -e "${2}${1}${NC}"
}