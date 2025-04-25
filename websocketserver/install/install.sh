#!/usr/bin/env bash
set -e

# -----------------------------------------------------------------------------
# Pastel Color Palette Setup using tput
# -----------------------------------------------------------------------------
if [ "$(tput colors)" -ge 256 ]; then
    # Using 256-color indices for pastel-like colors
    PASTEL_BLUE=$(tput setaf 153)    # Pastel blue
    PASTEL_GREEN=$(tput setaf 120)   # Pastel green
    PASTEL_PURPLE=$(tput setaf 176)  # Pastel purple
    PASTEL_ORANGE=$(tput setaf 215)  # Pastel orange
else
    # Fallback to basic colors if 256 colors not supported
    PASTEL_BLUE=$(tput setaf 4)
    PASTEL_GREEN=$(tput setaf 2)
    PASTEL_PURPLE=$(tput setaf 5)
    PASTEL_ORANGE=$(tput setaf 3)
fi
RESET=$(tput sgr0)


cat << 'EOF'                                                                                                
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                                                                                    
                                              ::::::::                                              
                                             ::::::::::                                             
                                            ::::::::::::                                            
                                           ::::::::::::::                                           
                                       :::::::::::::::::::::-                                       
                                    :::::::   :::::::    -------                                    
                                 ::::::       :::            -------                                
                      --------:::::-         ::::               --------------                      
                    ------------:            :::                   ------------                     
                    -----------             :::                      -----------                    
                    --------------:-        ::-                     ------------                    
                     --------- ----:::::-  :--                   =-------------                     
                       -------:      ---------                 -----    =----                       
                        --- ----          -------          -------       ---                        
                        ---   ----        --- ------     ------          --=                        
                        ---    ----      ---      ----------             ---                        
                        ---      ----   ----      -------=               ===                        
                        ---        ---- ---    -------------             ===                        
                        ---         --------------       -=--=           ===                        
                        ---           --------             ======        ===                        
                        ---          ---------                =====      ===                        
                        ---        -----    -----               =====    ===                        
                      =------   -------       ----===             ============                      
                     ===----------- ---          ========           ===========                     
                    ======-----     ---               ==========    ============                    
                    ========--=     ---                    =====================                    
                     =======------   --=                           ============                     
                       =====   ------=--=                      =======  =====                       
                                  ------===                ========                                 
                                      =======  ======   =======                                     
                                         ==================                                         
                                            ============                                            
                                            ===========                                             
                                             =========                                              
                                               ======                                               
                                                                                                    
                                                                                                    
                                                                                                    
                         ____  _     _        _ _           _           _ 
                        |  _ \(_)___| |_ _ __(_) |__  _   _| |_ ___  __| |
                        | | | | / __| __| '__| | '_ \| | | | __/ _ \/ _` |
                        | |_| | \__ \ |_| |  | | |_) | |_| | ||  __/ (_| |
                        |____/|_|___/\__|_|  |_|_.__/ \__,_|\__\___|\__,_|
                        | |/ /_ __   _____      _| | ___  __| | __ _  ___ 
                        | ' /| '_ \ / _ \ \ /\ / / |/ _ \/ _` |/ _` |/ _ \
                        | . \| | | | (_) \ V  V /| |  __/ (_| | (_| |  __/
                        |_|\_\_| |_|\___/ \_/\_/ |_|\___|\__,_|\__, |\___|
                                                               |___/                                                                                                          
                                                                                                                            
                                                                                                    
                                                                                                    
                                                                                                    
EOF

# -----------------------------------------------------------------------------
# 1. OS Detection
# -----------------------------------------------------------------------------
OS_TYPE=$(uname -s)

# -----------------------------------------------------------------------------
# 3. Installation Directories and Configuration Paths
# -----------------------------------------------------------------------------
INSTALL_PATH="/usr/local/bin"

read -p "${PASTEL_BLUE}MCP config file path${RESET} [default: ${PASTEL_GREEN}$HOME/.mcp.json${RESET}]: " MCP_CONFIG_DIR </dev/tty
MCP_CONFIG_DIR=${MCP_CONFIG_DIR:-$HOME/.mcp.json}


read -p "${PASTEL_BLUE}Syftbox config file path${RESET} [default: ${PASTEL_GREEN}$HOME/.syftbox/config.json${RESET}]: " SYFTBOX_CONFIG_DIR </dev/tty
SYFTBOX_CONFIG_DIR=${SYFTBOX_CONFIG_DIR:-$HOME/.syftbox/config.json}

# Check if Syftbox config directory exists
if [ ! -f "$SYFTBOX_CONFIG_DIR" ]; then
    SYFTBOX_CONFIG_DIR=""
fi

# -----------------------------------------------------------------------------
# 4. User Configuration: Credentials, Server, and Project Directories
# -----------------------------------------------------------------------------
# read -p "${PASTEL_BLUE}Enter your User ID${RESET}: " USER_ID </dev/tty
# if [ -z "$USER_ID" ]; then
#     echo "Error: User ID is required. Exiting."
#     exit 1
# fi

SERVER_ADDRESS="https://distributedknowledge.org"

# Prompt for User ID and check if it's already registered
while true; do
    read -p "${PASTEL_BLUE}Enter your User ID${RESET}: " USER_ID </dev/tty
    
    if [ -z "$USER_ID" ]; then
        echo "${PASTEL_RED}Error: User ID is required.${RESET}"
        continue
    fi
    
    echo "${PASTEL_BLUE}Checking if User ID is available...${RESET}"
    
    # Temporarily disable exit on error
    set +e
    
    # Check if userid exists directly
    check_endpoint="${SERVER_ADDRESS}/auth/check-userid/${USER_ID}"
    if command -v curl &>/dev/null; then
        response=$(curl -s "$check_endpoint")
    elif command -v wget &>/dev/null; then
        response=$(wget -q -O - "$check_endpoint")
    else
        echo "${PASTEL_RED}Error: Neither curl nor wget is available for checking User ID.${RESET}"
        exists_status=2
    fi
    
    # Re-enable exit on error
    set -e
    
    echo "Response: $response"
    
    # Determine if user exists
    if echo "$response" | grep -q '"exists": *true'; then
        echo "User ID '${USER_ID}' is already registered. Please choose another ID."
        continue
    elif echo "$response" | grep -q '"exists": *false'; then
        echo "User ID '${USER_ID}' is available. Continuing with installation..."
        break
    else
        echo "${PASTEL_RED}Error: Unexpected response from server.${RESET}"
        echo "Response: $response"
        read -p "${PASTEL_RED}Could not verify User ID. Do you want to try again? (y/n)${RESET}: " retry </dev/tty
        if [[ ! "$retry" =~ ^[Yy]$ ]]; then
            echo "${PASTEL_ORANGE}Proceeding with installation using User ID '${USER_ID}'.${RESET}"
            break
        fi
    fi
done

# Ensure USER_ID is set before continuing
if [ -z "$USER_ID" ]; then
    echo "${PASTEL_RED}Error: No User ID provided. Exiting.${RESET}"
    exit 1
fi




PROJECT_DIR="$HOME/.config/dk"
VECTOR_DB_DIR="$PROJECT_DIR/vector_db"
KEYS_PATH="$PROJECT_DIR/keys"

# -----------------------------------------------------------------------------
# 4A. Rag Source File Configuration
# -----------------------------------------------------------------------------
RAG_FILE="$PROJECT_DIR/rag_sources.jsonl"
mkdir -p "$PROJECT_DIR"

# -----------------------------------------------------------------------------
# 5. LLM Provider and Model Selection
# -----------------------------------------------------------------------------
echo ""
echo "${PASTEL_PURPLE}Select LLM provider:${RESET}"
echo "  1) OpenAI"
echo "  2) Anthropic"
echo "  3) Ollama [local]"
read -p "${PASTEL_BLUE}Choice${RESET} [1-3]: " choice </dev/tty
case "$choice" in
  1) LLM_PROVIDER="openai" ;;
  2) LLM_PROVIDER="anthropic" ;;
  3) LLM_PROVIDER="ollama" ;;
  *) echo "Invalid choice. Exiting." && exit 1 ;;
esac

if [ "$LLM_PROVIDER" == "openai" ]; then
    echo ""
    echo "${PASTEL_PURPLE}Select OpenAI model:${RESET}"
    echo "  1) gpt-4o-mini"
    echo "  2) gpt-4o"
    read -p "${PASTEL_BLUE}Choice${RESET} [1-2]: " choice </dev/tty
    case "$choice" in
      1) SELECTED_MODEL="gpt-4o-mini" ;;
      2) SELECTED_MODEL="gpt-4o" ;;
      *) echo "Invalid choice. Exiting." && exit 1 ;;
    esac
elif [ "$LLM_PROVIDER" == "anthropic" ]; then
    echo ""
    echo "${PASTEL_PURPLE}Select Anthropic model:${RESET}"
    echo "  1) claude-3.5-haiku"
    echo "  2) claude-3.7-sonnet"
    read -p "${PASTEL_BLUE}Choice${RESET} [1-2]: " choice </dev/tty
    case "$choice" in
      1) SELECTED_MODEL="claude-3.5-haiku" ;;
      2) SELECTED_MODEL="claude-3.7-sonnet" ;;
      *) echo "Invalid choice. Exiting." && exit 1 ;;
    esac
elif [ "$LLM_PROVIDER" == "ollama" ]; then
    echo ""
    echo "${PASTEL_PURPLE}Select Ollama model:${RESET}"
    echo "  1) gemma3:4b"
    echo "  2) qwen2.5:latest"
    echo "  3) deepseek-r1:7b"
    read -p "${PASTEL_BLUE}Choice${RESET} [1-3]: " choice </dev/tty
    case "$choice" in
      1) SELECTED_MODEL="gemma3:4b" ;;
      2) SELECTED_MODEL="qwen2.5:latest" ;;
      3) SELECTED_MODEL="deepseek-r1:7b" ;;
      *) echo "Invalid choice. Exiting." && exit 1 ;;
    esac
fi

# For OpenAI or Anthropic, read the API key directly into API_KEY
if [ "$LLM_PROVIDER" == "openai" ]; then
    read -p "${PASTEL_BLUE}Enter OPENAI_API_KEY (leave blank to use env variable)${RESET}: " API_KEY </dev/tty
elif [ "$LLM_PROVIDER" == "anthropic" ]; then
    read -p "${PASTEL_BLUE}Enter ANTHROPIC_API_KEY (leave blank to use env variable)${RESET}: " API_KEY </dev/tty
fi


echo ""
echo ""
echo ""
echo "${PASTEL_PURPLE}Installing / Pulling Ollama Models ${RESET}"
# -----------------------------------------------------------------------------
# 2. Dependency Check: Verify if 'ollama' is installed
# -----------------------------------------------------------------------------
if ! command -v ollama &>/dev/null; then
    echo "Dependency missing: 'ollama' is not installed."
    read -p "${PASTEL_BLUE}Install Ollama now?${RESET} [Y/n]: " install_ollama </dev/tty
    if [[ "$install_ollama" =~ ^[Yy]$ || -z "$install_ollama" ]]; then
        echo "Please visit https://ollama.ai/download to install Ollama, then re-run this script."
    fi
    exit 1
else
    echo "'ollama' is installed."
fi

echo "Pulling latest nomic-embed-text image from Ollama..."
ollama pull nomic-embed-text

if [ "$LLM_PROVIDER" == "ollama" ]; then
    echo "Pulling model '$SELECTED_MODEL' via Ollama..."
    ollama pull "$SELECTED_MODEL"
fi

# -----------------------------------------------------------------------------
# 6. Create Required Directories
# -----------------------------------------------------------------------------
mkdir -p "$PROJECT_DIR" "$VECTOR_DB_DIR" "$KEYS_PATH"

# -----------------------------------------------------------------------------
# 7. Generate MCP and Model Configuration Files
# -----------------------------------------------------------------------------
if [ "$LLM_PROVIDER" == "ollama" ]; then
    cat << EOF > "$PROJECT_DIR/model_config.json"
{
  "provider": "ollama",
  "base_url": "http://localhost:11434/api/generate",
  "model": "$SELECTED_MODEL",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
EOF
else
    cat << EOF > "$PROJECT_DIR/model_config.json"
{
  "provider": "$LLM_PROVIDER",
  "api_key": "$API_KEY",
  "model": "$SELECTED_MODEL",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
EOF
fi

if [ ! -f "$MCP_CONFIG_DIR" ]; then
    cat << EOF > "$MCP_CONFIG_DIR"
{
  "mcpServers": {
    "DistributedKnowledge": {
      "command": "$INSTALL_PATH/dk",
      "args": [
        "-userId", "$USER_ID",
        "-private", "$KEYS_PATH/private_key",
        "-public", "$KEYS_PATH/public_key",
        "-project_path", "$PROJECT_DIR",
        "-syftbox_config", "$SYFTBOX_CONFIG_DIR",
        "-rag_sources", "$RAG_FILE",
        "-server", "$SERVER_ADDRESS"
      ]
    }
  }
}
EOF
else
    if grep -q '"mcpServers"' "$MCP_CONFIG_DIR"; then
        awk -v ip="$INSTALL_PATH/dk" \
            -v uid="$USER_ID" \
            -v syftbox="$SYFTBOX_CONFIG_DIR" \
            -v kp="$KEYS_PATH" \
            -v pd="$PROJECT_DIR" \
            -v rg="$RAG_FILE" \
            -v sa="$SERVER_ADDRESS" '
BEGIN { inBlock=0; inserted=0 }
/"mcpServers"[[:space:]]*:[[:space:]]*{/ { inBlock=1 }
{
  if (inBlock && /^[[:space:]]*}\s*$/ && inserted==0) {
    print "    },\"DistributedKnowledge\": {"
    print "        \"command\": \"" ip "\","
    print "        \"args\": ["
    print "            \"-userId\", \"" uid "\","
    print "            \"-syftbox_config\", \"" syftbox "\","
    print "            \"-private\", \"" kp "/private_key\","
    print "            \"-public\", \"" kp "/public_key\","
    print "            \"-project_path\", \"" pd "\","
    print "            \"-rag_sources\", \"" rg "\","
    print "            \"-server\", \"" sa "\""
    print "        ]"
    print "    }"
    inserted=1
  }
  print
  if (inBlock && /^[[:space:]]*}\s*$/) { inBlock=0 }
}' "$MCP_CONFIG_DIR" > "$MCP_CONFIG_DIR.tmp" && mv "$MCP_CONFIG_DIR.tmp" "$MCP_CONFIG_DIR" && sed -i '${/^[[:space:]]*}$/d;}' "$MCP_CONFIG_DIR" 
    else
        cat << EOF > "$MCP_CONFIG_DIR.tmp"
{
  "mcpServers": {
    "DistributedKnowledge": {
      "command": "$INSTALL_PATH/dk",
      "args": [
        "-userId", "$USER_ID",
        "-private", "$KEYS_PATH/private_key",
        "-public", "$KEYS_PATH/public_key",
        "-project_path", "$PROJECT_DIR",
        "-syftbox_config", "$SYFTBOX_CONFIG_DIR",
        "-rag_sources", "$RAG_FILE",
        "-server", "$SERVER_ADDRESS"
      ]
    }
  },
  "existingConfig": $(cat "$MCP_CONFIG_DIR")
}
EOF
        mv "$MCP_CONFIG_DIR.tmp" "$MCP_CONFIG_DIR"
    fi
fi


# -----------------------------------------------------------------------------
# 8. OS-Specific Post-Installation: Download DK Executable
# -----------------------------------------------------------------------------
if [ "$OS_TYPE" == "Darwin" ]; then
    if command -v curl &>/dev/null; then
        sudo curl -fsSL "https://distributedknowledge.org/download/mac" -o "$INSTALL_PATH/dk"
    elif command -v wget &>/dev/null; then
        sudo wget -q "https://distributedknowledge.org/download/mac" -O "$INSTALL_PATH/dk"
    else
        echo "Error: Neither curl nor wget is available." >&2
        exit 1
    fi
elif [ "$OS_TYPE" == "Linux" ]; then
    if command -v curl &>/dev/null; then
        sudo curl -fsSL "https://distributedknowledge.org/download/linux" -o "$INSTALL_PATH/dk"
    elif command -v wget &>/dev/null; then
        sudo wget -q "https://distributedknowledge.org/download/linux" -O "$INSTALL_PATH/dk"
    else
        echo "Error: Neither curl nor wget is available." >&2
        exit 1
    fi
fi


sudo chmod +x "$INSTALL_PATH/dk"


echo ""
echo ""
echo ""
echo ""
echo "$PASTEL_PURPLE Configuration $RESET"
echo ""
echo "$PASTEL_GREEN User ID: $PASTEL_ORANGE $USER_ID"
echo "$PASTEL_GREEN Server: $PASTEL_ORANGE $SERVER_ADDRESS"
echo "$PASTEL_GREEN Project Directory: $PASTEL_ORANGE $PROJECT_DIR"
echo "$PASTEL_GREEN Model: $PASTEL_ORANGE $SELECTED_MODEL"
echo "$RESET"

cat << 'EOF'

·····························································································································································
:  ____    _  __    ___                 _             _   _           _     _                      ____                               _          _          :
: |  _ \  | |/ /   |_ _|  _ __    ___  | |_    __ _  | | | |   __ _  | |_  (_)   ___    _ __      / ___|   ___    _ __ ___    _ __   | |   ___  | |_    ___ :
: | | | | | ' /     | |  | '_ \  / __| | __|  / _` | | | | |  / _` | | __| | |  / _ \  | '_ \    | |      / _ \  | '_ ` _ \  | '_ \  | |  / _ \ | __|  / _ \:
: | |_| | | . \     | |  | | | | \__ \ | |_  | (_| | | | | | | (_| | | |_  | | | (_) | | | | |   | |___  | (_) | | | | | | | | |_) | | | |  __/ | |_  |  __/:
: |____/  |_|\_\   |___| |_| |_| |___/  \__|  \__,_| |_| |_|  \__,_|  \__| |_|  \___/  |_| |_|    \____|  \___/  |_| |_| |_| | .__/  |_|  \___|  \__|  \___|:
:                                                                                                                            |_|                            :
·····························································································································································

EOF
