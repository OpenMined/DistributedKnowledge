#!/usr/bin/env bash
set -e

# Determine the OS type
OS_TYPE=$(uname -s)
echo "Detected OS: $OS_TYPE"

# -----------------------------------------------------------------------------
# 1. Environment Check - Verify if 'ollama' is installed, prompt to install if not.
# -----------------------------------------------------------------------------
# (ollama installation code remains unchanged...)

# -----------------------------------------------------------------------------
# 2. System Recognition and Basic Installation Directories
# -----------------------------------------------------------------------------
# Prompt for installation directory for the binary (default: /usr/local/bin)
DEFAULT_INSTALL_PATH="/usr/local/bin"
read -p "Enter the installation directory for Distributed Knowledge binary [default: $DEFAULT_INSTALL_PATH]: " INSTALL_PATH </dev/tty
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
read -p "Enter the Server Address [default: $DEFAULT_SERVER_ADDRESS]: " SERVER_ADDRESS </dev/tty
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
KEYS_PATH="$PROJECT_DIR/keys"

# -----------------------------------------------------------------------------
# 3A. Rag Source File Configuration
# -----------------------------------------------------------------------------
# New behavior: if the user does not specify a Rag source file, default to $PROJECT_DIR/rag_sources.jsonl.
# The prompt now includes a comment explaining that if left blank, a sample rag source will be generated.
DEFAULT_RAG_FILE="$PROJECT_DIR/rag_sources.jsonl"
mkdir -p $PROJECT_DIR
read -p "Enter the path for Rag source file [default: $DEFAULT_RAG_FILE] (if left blank, a sample rag source will be generated): " RAG_DIR </dev/tty
if [ -z "$RAG_DIR" ]; then
    RAG_DIR="$DEFAULT_RAG_FILE"
    echo "No Rag source file provided. Creating sample file at $RAG_DIR..."
    # Embedded sample content from your provided rag_sources.jsonl file:
    cat << 'EOF' > "$RAG_DIR"
{
  "file": "relationships_questionnaire.md",
  "text": "# Relationships Questionnaire\n\n1. **What qualities do you value in a friend?**\n   I value honesty, loyalty, empathy, and a good sense of humor in a friend.\n\n2. **How do you build and maintain strong relationships?**\n   Through open communication, mutual respect, and spending quality time together.\n\n3. **Who has been your greatest mentor or role model?**\n   My greatest mentor has been Sarah, who taught me the importance of resilience and persistence.\n\n4. **What do you look for in a partner?**\n   I look for someone who shares my values, is kind, and intellectually stimulating.\n\n5. **How do you resolve conflicts with loved ones?**\n   By communicating openly, listening, and finding a compromise.\n\n6. **How do you show appreciation to others?**\n   Through small gestures, kind words, and spending quality time together.\n\n7. **What role does trust play in your relationships?**\n   Trust is the foundation of any strong relationship, ensuring honesty and emotional safety.\n\n8. **How have your relationships shaped you?**\n   They’ve taught me empathy, patience, and the value of surrounding myself with supportive people.\n\n9. **What is a lesson you've learned from a past relationship?**\n   Communication is key to avoiding misunderstandings and resolving issues.\n\n10. **How do you support others emotionally?**\n   By listening, offering reassurance, and providing comfort when needed."
}
{
  "file": "career_aspirations_questionnaire.md",
  "text": "# Career Aspirations Questionnaire\n\n1. **What career path are you currently pursuing?**\n   I’m pursuing a career in digital marketing, focusing on content strategy and brand development.\n\n2. **What inspired you to choose this field?**\n   A marketing campaign I saw during college really inspired me, as it showed the power of storytelling in connecting with an audience.\n\n3. **What skills do you want to develop professionally?**\n   I want to enhance my leadership and project management skills, especially in large-scale campaigns.\n\n4. **What does success mean to you in your career?**\n   Success is about achieving personal growth, building meaningful relationships, and making a tangible impact through my work.\n\n5. **What challenges have you faced in your career journey?**\n   I’ve faced challenges like adapting to fast-changing industry trends, but it’s taught me how to stay agile and always keep learning.\n\n6. **How do you stay motivated at work?**\n   I stay motivated by setting goals, celebrating small wins, and focusing on the bigger picture.\n\n7. **What role does passion play in your career decisions?**\n   Passion drives me to give my best effort and stay committed to my goals, even during difficult times.\n\n8. **Where do you see yourself in five years?**\n   I see myself in a senior marketing role, leading a team and driving major projects that impact the company’s growth.\n\n9. **Who is someone in your field you admire?**\n   I admire Emma Thompson, whose innovative campaigns have changed how brands engage with consumers globally.\n\n10. **What is your proudest career accomplishment so far?**\n   My proudest accomplishment was leading a project that increased our company’s social media engagement by 40%."
}
{
  "file": "creativity_and_expression_questionnaire.md",
  "text": "# Creativity and Expression Questionnaire\n\n1. **What does creativity mean to you?**\n   Creativity is about thinking outside the box, generating new ideas, and finding innovative ways to express myself.\n\n2. **In what ways do you express your creativity?**\n   I express it through writing and photography, often capturing moments that tell a deeper story.\n\n3. **Do you prefer structured or spontaneous creative processes?**\n   I lean toward spontaneity but appreciate structure for complex projects that require detailed planning.\n\n4. **Who inspires your creative work?**\n   I’m inspired by artists like Frida Kahlo and filmmakers like Wes Anderson, who use their work to convey deep emotions.\n\n5. **What is your favorite creative project you’ve done?**\n   My favorite project was creating a photo series on city life, where I explored contrasts between old architecture and modern culture.\n\n6. **How do you deal with creative blocks?**\n   I step away from the project, go for a walk, or engage with other forms of art to get the creative juices flowing again.\n\n7. **What role does creativity play in your everyday life?**\n   Creativity helps me solve problems, stay inspired, and approach everyday tasks in new and interesting ways.\n\n8. **Do you prefer to share your work or keep it private?**\n   I enjoy sharing my work with others to get feedback, but I also like to keep personal projects private for myself.\n\n9. **What medium do you feel most comfortable with (writing, art, music, etc.)?**\n   I feel most comfortable with photography, as it allows me to visually express stories and emotions.\n\n10. **How do you nurture and grow your creativity?**\n   By trying new hobbies, seeking inspiration from different cultures, and pushing myself to step outside my comfort zone."
}
{
  "file": "health_and_wellness_questionnaire.md",
  "text": "# Health and Wellness Questionnaire\n\n1. **How do you define wellness in your life?**\n   Wellness is a holistic approach to well-being that includes physical fitness, mental clarity, and emotional balance.\n\n2. **What does a healthy day look like for you?**\n   A healthy day includes morning exercise, balanced meals, taking time to relax, and getting plenty of sleep.\n\n3. **How do you stay physically active?**\n   I stay active with a combination of running, yoga, and weightlifting.\n\n4. **What role does mental health play in your life?**\n   Mental health is essential for maintaining overall well-being and helping me handle daily challenges.\n\n5. **What habits help you sleep and rest well?**\n   I follow a bedtime routine where I avoid screens and practice mindfulness before going to sleep.\n\n6. **How do you handle illness or setbacks?**\n   I focus on taking care of myself, staying patient, and seeking support when necessary to recover quickly.\n\n7. **Do you follow a specific diet or nutrition plan?**\n   I follow a balanced diet with lots of vegetables, lean proteins, and whole grains.\n\n8. **How do you manage screen time and digital wellness?**\n   I set daily limits for screen time and take frequent breaks to reduce strain on my eyes and mind.\n\n9. **What activities help you unwind?**\n   I unwind by reading, practicing meditation, or going for a nature walk.\n\n10. **How do you set and maintain health goals?**\n   I set specific, measurable goals and track my progress regularly to stay motivated."
}
{
  "file": "dreams_and_imagination_questionnaire.md",
  "text": "# Dreams and Imagination Questionnaire\n\n1. **What’s one dream you’ve had since childhood?**\n   Since childhood, I’ve dreamed of traveling the world, experiencing new cultures, and documenting my journey through photography.\n\n2. **What role does imagination play in your life?**\n   Imagination fuels my creativity, helping me to see beyond the obvious and find new solutions to problems.\n\n3. **If you could live any alternate life, what would it look like?**\n   I’d live a life as a traveling photographer, capturing the beauty of the world while experiencing different cultures firsthand.\n\n4. **What’s the most fantastical idea you've ever thought of?**\n   One of the most fantastical ideas I’ve had was creating a self-sustaining city in the middle of a forest that uses renewable energy.\n\n5. **Have any of your dreams influenced real-life actions?**\n   Yes, my dream of becoming a photographer inspired me to pursue photography seriously and turn it into a career.\n\n6. **What inspires your daydreams?**\n   Daydreams are inspired by nature, adventure, and the idea of exploring unknown places.\n\n7. **What’s something impossible you wish could happen?**\n   I wish humans could live sustainably on Mars, making space travel a common part of everyday life.\n\n8. **Do you believe dreams have meanings?**\n   I believe dreams can offer insights into our subconscious and reflect our deeper emotions or desires.\n\n9. **How do you tap into your imagination?**\n   I tap into it by reading, listening to music, and exploring new experiences that push my boundaries.\n\n10. **What future do you envision for yourself and the world?**\n   I envision a future where I’ve traveled the world, learned new skills, and contributed to environmental sustainability through my work."
}
EOF
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
# mkdir -p "$PROJECT_DIR" "$QUERIES_DIR" "$ANSWERS_DIR" "$DEFAULT_VECTOR_DB_DIR" "$KEYS_PATH"

# -----------------------------------------------------------------------------
# 6. Install the Binary and Set Permissions
# -----------------------------------------------------------------------------
echo "Installing the binary..."
# cp myapp "$INSTALL_PATH/"
# chmod +x "$INSTALL_PATH/myapp"

# -----------------------------------------------------------------------------
# 7. Set Up the MCP Configuration File
# -----------------------------------------------------------------------------
echo "Setting up MCP configuration file..."
echo "User ID: $USER_ID"
echo "Server Address: $SERVER_ADDRESS"
echo "Project Directory: $PROJECT_DIR"
echo "Queries Directory: $QUERIES_DIR"
echo "Answers Directory: $ANSWERS_DIR"
echo "Keys Path: $KEYS_PATH"
echo "Rag Source: $RAG_DIR"

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
    echo "MCP configuration file does not exist. Creating a new one..."
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
    echo "MCP configuration file already exists. Adding new DistributedKnowledge entry..."
    if grep -q '"mcpServers"' "$MCP_CONFIG_DIR"; then
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
        "-rag_sources", "$RAG_DIR",
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
echo "MCP configuration file updated successfully."


chmod +x "$INSTALL_PATH/dk"

# -----------------------------------------------------------------------------
# 9. OS-specific Post-Installation Steps
# -----------------------------------------------------------------------------
if [ "$OS_TYPE" == "Darwin" ]; then
    # -----------------------------------------------------------------------------
    # 9. Download / Install DK executable
    # -----------------------------------------------------------------------------
    echo "Installing the Distributed Knowledge App..."
    if command -v curl > /dev/null 2>&1; then
        echo "Using curl to download the binary."
        curl -fsSL "https://distributedknowledge.org/download/mac" -o "$INSTALL_PATH/dk"
    elif command -v wget > /dev/null 2>&1; then
        echo "Using wget to download the binary."
        wget -q "https://distributedknowledge.org/download/mac" -O "$INSTALL_PATH/dk"
    else
        echo "Error: Neither curl nor wget is installed." >&2
        exit 1
    fi
elif [ "$OS_TYPE" == "Linux" ]; then
    # -----------------------------------------------------------------------------
    # 9. Download / Install DK executable
    # -----------------------------------------------------------------------------
    echo "Installing the Distributed Knowledge App..."
    if command -v curl > /dev/null 2>&1; then
        echo "Using curl to download the binary."
        curl -fsSL "https://distributedknowledge.org/download/linux" -o "$INSTALL_PATH/dk"
    elif command -v wget > /dev/null 2>&1; then
        echo "Using wget to download the binary."
        wget -q "https://distributedknowledge.org/download/linux" -O "$INSTALL_PATH/dk"
    else
        echo "Error: Neither curl nor wget is installed." >&2
        exit 1
    fi
fi

echo "Installation complete!"
