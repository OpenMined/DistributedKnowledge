#!/usr/bin/env bash
set -e

# Determine the OS type
OS_TYPE=$(uname -s)
echo "Detected OS: $OS_TYPE"

# -----------------------------------------------------------------------------
# 1. Environment Check - Verify if 'ollama' is installed, prompt to install if not.
# -----------------------------------------------------------------------------
# if ! command -v ollama > /dev/null 2>&1; then
#     echo "Warning: 'ollama' is not installed on this system."
#     read -p "Would you like to install 'ollama' from the trusted source? (y/n): " install_ollama
#     if [[ "$install_ollama" =~ ^[Yy]$ ]]; then
#         echo "Installing 'ollama'..."
#         if [ "$OS_TYPE" == "Darwin" ]; then
#             # macOS installation command for ollama (replace URL/command with the trusted source)
#             curl -fsSL https://ollama.example.com/install.sh | bash
#         elif [ "$OS_TYPE" == "Linux" ]; then
#             # Linux installation command for ollama (replace URL/command with the trusted source)
#             curl -fsSL https://ollama.example.com/install.sh | sudo bash
#         else
#             echo "Unsupported OS for automatic ollama installation. Please install ollama manually."
#             exit 1
#         fi
#         echo "'ollama' installation completed."
#     else
#         echo "'ollama' installation declined. Exiting installation process."
#         exit 1
#     fi
# fi



# -----------------------------------------------------------------------------
# 2. System Recognition and Basic Installation Directories
# -----------------------------------------------------------------------------
# Prompt for installation directory for the binary (default: /usr/local/bin)
DEFAULT_INSTALL_PATH="/usr/local/bin"
read -p "Enter the installation directory for MyApp binary [default: $DEFAULT_INSTALL_PATH]: " INSTALL_PATH </dev/tty
if [ -z "$INSTALL_PATH" ]; then
    INSTALL_PATH="$DEFAULT_INSTALL_PATH"
fi

# MCP configuration path (default: ~/.mcp.json)
DEFAULT_MCP_CONFIG_DIR="$HOME/.mcp.json"
read -p "Enter the directory for the MCP config file [default: $DEFAULT_MCP_CONFIG_DIR]: " MCP_CONFIG_DIR </dev/tty
if [ -z "$MCP_CONFIG_DIR" ]; then
    MCP_CONFIG_DIR="$DEFAULT_MCP_CONFIG_DIR"
fi

# -----------------------------------------------------------------------------
# 3. User Input Prompts for Detailed Configuration
# -----------------------------------------------------------------------------
# User credentials and server details
read -p "Enter your User ID: " USER_ID </dev/tty
if [ -z "$USER_ID" ]; then
    echo "Error: User ID is required. Exiting."
    exit 1
fi

DEFAULT_SERVER_ADDRESS="https://distributedknowledge.org"
read -p "Enter the Server Address: [default: $DEFAULT_SERVER_ADDRESS]" SERVER_ADDRESS </dev/tty
if [ -z "$SERVER_ADDRESS" ]; then
    SERVER_ADDRESS="$DEFAULT_SERVER_ADDRESS"
fi


# Project Directory (must be provided)
DEFAULT_PROJECT_DIR="$HOME/.config"
read -p "Enter the Project Directory [default: $HOME/.config]: " PROJECT_DIR </dev/tty
if [ -z "$PROJECT_DIR" ]; then
    PROJECT_DIR="$DEFAULT_PROJECT_DIR"
fi
PROJECT_DIR+="/dk"

# Project sub-directories
QUERIES_DIR="$PROJECT_DIR/queries.json"
ANSWERS_DIR="$PROJECT_DIR/answers.json"
AUTO_APPROVAL="$PROJECT_DIR/auto_approval.json"

DEFAULT_VECTOR_DB_DIR="$PROJECT_DIR/vector_db"

# PUBLIC/PRIVATE Keys Path
KEYS_PATH="$PROJECT_DIR/keys"

# Directory for Rag source files
DEFAULT_RAG_DIR="$PROJECT_DIR/rag"
read -p "Enter the directory for Rag source files [default: $DEFAULT_RAG_DIR]: " RAG_DIR </dev/tty
if [ -z "$RAG_DIR" ]; then
    RAG_DIR="$DEFAULT_RAG_DIR"
fi

# -----------------------------------------------------------------------------
# 4. LLM Provider, Model Selection, and API Key Configuration
# -----------------------------------------------------------------------------
echo ""
echo "Select the LLM provider for generating answers:"
echo "1) OpenAI"
echo "2) Anthropic"
echo "3) Ollama [local]"
read -p "Enter your choice [1-3]: " LLM_PROVIDER_CHOICE </dev/tty

case "$LLM_PROVIDER_CHOICE" in
  1)
    LLM_PROVIDER="openai"
    ;;
  2)
    LLM_PROVIDER="anthropic"
    ;;
  3)
    LLM_PROVIDER="ollama"
    ;;
  *)
    echo "Invalid choice. Exiting."
    exit 1
    ;;
esac

if [ "$LLM_PROVIDER" == "openai" ]; then
    echo ""
    echo "Select an OpenAI model:"
    echo "1) gpt-4o-mini"
    echo "2) gpt-4o"
    read -p "Enter your choice [1-2]: " MODEL_CHOICE </dev/tty
    case "$MODEL_CHOICE" in
      1)
        SELECTED_MODEL="gpt-4o-mini"
        ;;
      2)
        SELECTED_MODEL="gpt-4o"
        ;;
      *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
    esac
elif [ "$LLM_PROVIDER" == "anthropic" ]; then
    echo ""
    echo "Select an Anthropic model:"
    echo "1) claude-3.5-haiku"
    echo "2) claude-3.7-sonnet"
    read -p "Enter your choice [1-2]: " MODEL_CHOICE </dev/tty
    case "$MODEL_CHOICE" in
      1)
        SELECTED_MODEL="claude-3.5-haiku"
        ;;
      2)
        SELECTED_MODEL="claude-3.7-sonnet"
        ;;
      *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
    esac
elif [ "$LLM_PROVIDER" == "ollama" ]; then
    echo ""
    echo "Select an Ollama model:"
    echo "1) gemma3:4b"
    echo "2) qwen2.5:latest"
    echo "3) deepseek-r1:7b"
    read -p "Enter your choice [1-3]: " MODEL_CHOICE </dev/tty
    case "$MODEL_CHOICE" in
      1)
        SELECTED_MODEL="gemma3:4b"
        ;;
      2)
        SELECTED_MODEL="qwen2.5:latest"
        ;;
      3)
        SELECTED_MODEL="deepseek-r1:7b"
        ;;
      *)
        echo "Invalid choice. Exiting."
        exit 1
        ;;
    esac
fi

# For OpenAI or Anthropic, prompt for the API Key. (If left empty, the corresponding environment variable will be used.)
if [ "$LLM_PROVIDER" == "openai" ]; then
    read -p "Enter your OPENAI_API_KEY (if empty, the environment variable will be used): " OPENAI_API_KEY </dev/tty
    API_KEY="$OPENAI_API_KEY"
elif [ "$LLM_PROVIDER" == "anthropic" ]; then
    read -p "Enter your ANTHROPIC_API_KEY (if empty, the environment variable will be used): " ANTHROPIC_API_KEY </dev/tty
    API_KEY="$ANTHROPIC_API_KEY"
fi

# -----------------------------------------------------------------------------
# 5. Create Necessary Directories
# -----------------------------------------------------------------------------
echo "Creating required directories..."
# Directories that might need elevated permissions
# sudo mkdir -p "$INSTALL_PATH"
# sudo mkdir -p "$MCP_CONFIG_DIR"
# User-specific project directories
# mkdir -p "$PROJECT_DIR" "$QUERIES_DIR" "$ANSWERS_DIR" "$DEFAULT_VECTOR_DB_DIR" "$KEYS_PATH" "$RAG_DIR"

# -----------------------------------------------------------------------------
# 6. Install the Binary and Set Permissions
# -----------------------------------------------------------------------------
echo "Installing the binary..."
# sudo cp myapp "$INSTALL_PATH/"
# sudo chmod +x "$INSTALL_PATH/myapp"

# -----------------------------------------------------------------------------
# 7. Set Up the MCP Configuration File
# -----------------------------------------------------------------------------
echo "Setting up MCP configuration file..."
echo "User ID: $USER_ID"
echo "Server Address: $SERVER_ADDRESS"
echo "Project Directory: $PROJECT_DIR"
echo "Queries Directory: $QUERIES_DIR"
echo "Answers Directory: $ANSWERS_DIR"
# echo "Automatic Approval: $AUTO_APPROVAL"
# echo "Vector DB Directory: $DEFAULT_VECTOR_DB_DIR"
echo "Keys Path: $KEYS_PATH"
echo "Rag Source Directory: $RAG_DIR"

# -----------------------------------------------------------------------------
# 8. Generate LLM Model Configuration JSON
# -----------------------------------------------------------------------------
echo "Creating LLM model configuration file at $PROJECT_DIR/model_config.json..."

mkdir -p "$PROJECT_DIR"
mkdir -p "$KEYS_PATH"

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
echo "Model configuration file created successfully."

# -----------------------------------------------------------------------------
# 8.5. Update MCP Configuration File
# -----------------------------------------------------------------------------
echo "Updating MCP configuration file at $MCP_CONFIG_DIR..."
if [ ! -f "$MCP_CONFIG_DIR" ]; then
    echo "Error: MCP configuration file '$MCP_CONFIG_DIR' does not exist."
    exit 1
fi

MCP_CONTENT=$(cat "$MCP_CONFIG_DIR")

# Check if the file is an empty JSON object (either completely empty or with an empty mcpServers)
if [[ "$MCP_CONTENT" =~ ^\{\s*\}$ ]] || [[ "$MCP_CONTENT" =~ \"mcpServers\"[[:space:]]*:[[:space:]]*\{\s*\} ]]; then
    echo "MCP configuration file is empty. Filling with default configuration..."
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
        "-rag_sources", "$RAG_DIR",
        "-server", "$SERVER_ADDRESS"
      ]
    }
  }
}
EOF
else
    echo "MCP configuration file already contains data. Adding new DistributedKnowledge entry..."
    if grep -q '"mcpServers"' "$MCP_CONFIG_DIR"; then
        # Use an awk script to insert the new key into the existing mcpServers object.
        awk -v install_path="$INSTALL_PATH/dk" \
            -v user_id="$USER_ID" \
            -v keys_path="$KEYS_PATH" \
            -v project_dir="$PROJECT_DIR" \
            -v rag_dir="$RAG_DIR" \
            -v server_address="$SERVER_ADDRESS" '
BEGIN { inBlock=0; inserted=0 }
/"mcpServers"[[:space:]]*:[[:space:]]*{/ { inBlock=1 }
{
  if (inBlock && /^[[:space:]]*}\s*$/ && inserted==0) {
    # Insert with a preceding comma (assuming the block already had at least one entry)
    print "    },\"DistributedKnowledge\": {"
    print "        \"command\": \"" install_path "\","
    print "        \"args\": ["
    print "            \"-userId\", \"" user_id "\","
    print "            \"-private\", \"" keys_path "/private_key\","
    print "            \"-public\", \"" keys_path "/public_key\","
    print "            \"-project_path\", \"" project_dir "\","
    print "            \"-rag_sources\", \"" rag_dir "\","
    print "            \"-server\", \"" server_address "\""
    print "        ]"
    print "    }"
    inserted=1
  }
  print
  if (inBlock && /^[[:space:]]*}\s*$/) { inBlock=0 }
}
' "$MCP_CONFIG_DIR" > "$MCP_CONFIG_DIR.tmp" && mv "$MCP_CONFIG_DIR.tmp" "$MCP_CONFIG_DIR"
    else
        echo "No \"mcpServers\" key found in MCP configuration file. Adding one..."
        # Wrap the existing configuration in a new object under mcpServers.
        cat << EOF > "$MCP_CONFIG_DIR.tmp"
{
  "mcpServers": {
    "DistributedKnowledge": {
      "command": "$INSTALL_PATH"/dk,
      "args": [
        "-userId", "$USER_ID",
        "-private", "$KEYS_PATH/private_key",
        "-public", "$KEYS_PATH/public_key",
        "-project_path", "$PROJECT_DIR",
        "-rag_sources", "$RAG_DIR",
        "-server", "$SERVER_ADDRESS"
      ]
    }
  },
  "existingConfig": $MCP_CONTENT
}
EOF
        mv "$MCP_CONFIG_DIR.tmp" "$MCP_CONFIG_DIR"
    fi
fi
echo "MCP configuration file updated successfully."


# -----------------------------------------------------------------------------
# 9. Download / Install DK executable
# -----------------------------------------------------------------------------

echo "Installing the Distributed Knowledge App..."
if command -v curl > /dev/null 2>&1; then
    echo "Using curl to download the binary."
    curl -fsSL "https://distributedknowledge.org/download" -o "$INSTALL_PATH/dk"
elif command -v wget > /dev/null 2>&1; then
    echo "Using wget to download the binary."
    wget -q "https://distributedknowledge.org/download" -O "$INSTALL_PATH/dk"
else
    echo "Error: Neither curl nor wget is installed." >&2
    exit 1
fi

chmod +x "$INSTALL_PATH/dk"

# -----------------------------------------------------------------------------
# 9. OS-specific Post-Installation Steps
# -----------------------------------------------------------------------------
if [ "$OS_TYPE" == "Darwin" ]; then
    echo "Running macOS-specific steps..."
    # Example: Provide guidance to add the installation directory to PATH
    # echo "Consider adding $INSTALL_PATH to your PATH in your shell profile."
elif [ "$OS_TYPE" == "Linux" ]; then
    echo "Running Linux-specific steps..."
    # Example: Install systemd service if needed
    # sudo cp myapp.service /etc/systemd/system/
    # sudo systemctl daemon-reload
fi

echo "Installation complete!"
