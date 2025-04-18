# Getting Started with Distributed Knowledge

This guide will help you set up and run your own instance of Distributed Knowledge, connecting to the network and leveraging collective intelligence for your AI applications.

!!! Prerequisites

    Before getting started, make sure you have the following installed:

    - **Ollama with `nomic-embed-text` model**: Required for local RAG vector embeddings
    - **Access to LLM providers**: You'll need API access to at least one of:
      - Anthropic (Claude)
      - OpenAI (GPT models)
      - Ollama (for local LLM hosting)

## Installation

=== "macOS and Linux"
    

    !!! Prerequisites
        - `curl` or `wget` installed for downloading files.
        - `ollama` CLI installed (for managing local LLM models, generate local RAG).

    Run the following command to start the installation process:
    ```bash
    curl -sSL https://distributedknowledge.org/install.sh | bash
    ```


=== "Windows"
    !!! Prerequisites
        - `ollama` CLI installed (for managing local LLM models, generate local RAG).

    Visit [distributedknowledge.org/downloads](https://distributedknowledge.org/downloads) to download DK Installer.



That's it! You now have DK embedded in your favorite LLM! 


## Troubleshooting

- Ensure `curl` or `wget` is installed for downloads.
- Install `ollama` if you plan to use local LLM models.
- Verify your User ID is unique and correctly entered.
- Provide valid API keys for OpenAI or Anthropic if required.
- Run the script with sufficient permissions to write to `/usr/local/bin`.

---

## Additional Resources

- [Distributed Knowledge Website](https://distributedknowledge.org)
- [Ollama Download](https://ollama.ai/download)
- [OpenAI API Documentation](https://platform.openai.com/docs)
- [Anthropic API Documentation](https://docs.anthropic.com)

---


## Next Steps

- Learn about [Architecture Concepts](../architecture/overview.md)
- Explore [Advanced Configuration](../configuration/advanced.md)
- Check out our [Tutorials](../tutorials/basic_usage.md) for common use cases
- Join the community on [GitHub](https://github.com/OpenMined/DistributedKnowledge)
