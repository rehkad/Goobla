<div align="center">
  <a href="https://goobla.com">
    <img alt="Goobla" height="200px" src="https://github.com/goobla/goobla/assets/3325447/0d0b44e2-8f4a-4e99-9b52-a5c1c741c8f7">
  </a>
</div>

# Goobla

Get up and running with large language models.

### macOS

[Download](https://goobla.com/download/Goobla-darwin.zip)

### Windows

[Download](https://goobla.com/download/GooblaSetup.exe)

### Homebrew (macOS & Linux)

```shell
brew install goobla
```

### Linux

```shell
curl -fsSL https://goobla.com/install.sh | sh
```
> [!WARNING]
> Inspect the script or verify its checksum before running. You can download the script from [install.sh](https://github.com/goobla/goobla/blob/main/scripts/install.sh) to review it first.

[Manual install instructions](https://github.com/goobla/goobla/blob/main/docs/linux.md)

### Docker

The official [Goobla Docker image](https://hub.docker.com/r/goobla/goobla) `goobla/goobla` is available on Docker Hub.

### Libraries

- [goobla-python](https://github.com/goobla/goobla-python)
- [goobla-js](https://github.com/goobla/goobla-js)

### Community

- [Discord](https://discord.gg/goobla)
- [Reddit](https://reddit.com/r/goobla)

## Quickstart

To run and chat with [Gemma 3](https://goobla.com/library/gemma3):

```shell
goobla run gemma3
```

## Model library

Goobla supports a list of models available on [goobla.com/library](https://goobla.com/library 'goobla model library')

Here are some example models that can be downloaded:

| Model              | Parameters | Size  | Download                         |
| ------------------ | ---------- | ----- | -------------------------------- |
| Gemma 3            | 1B         | 815MB | `goobla run gemma3:1b`           |
| Gemma 3            | 4B         | 3.3GB | `goobla run gemma3`              |
| Gemma 3            | 12B        | 8.1GB | `goobla run gemma3:12b`          |
| Gemma 3            | 27B        | 17GB  | `goobla run gemma3:27b`          |
| QwQ                | 32B        | 20GB  | `goobla run qwq`                 |
| DeepSeek-R1        | 7B         | 4.7GB | `goobla run deepseek-r1`         |
| DeepSeek-R1        | 671B       | 404GB | `goobla run deepseek-r1:671b`    |
| Llama 4            | 109B       | 67GB  | `goobla run llama4:scout`        |
| Llama 4            | 400B       | 245GB | `goobla run llama4:maverick`     |
| Llama 3.3          | 70B        | 43GB  | `goobla run llama3.3`            |
| Llama 3.2          | 3B         | 2.0GB | `goobla run llama3.2`            |
| Llama 3.2          | 1B         | 1.3GB | `goobla run llama3.2:1b`         |
| Llama 3.2 Vision   | 11B        | 7.9GB | `goobla run llama3.2-vision`     |
| Llama 3.2 Vision   | 90B        | 55GB  | `goobla run llama3.2-vision:90b` |
| Llama 3.1          | 8B         | 4.7GB | `goobla run llama3.1`            |
| Llama 3.1          | 405B       | 231GB | `goobla run llama3.1:405b`       |
| Phi 4              | 14B        | 9.1GB | `goobla run phi4`                |
| Phi 4 Mini         | 3.8B       | 2.5GB | `goobla run phi4-mini`           |
| Mistral            | 7B         | 4.1GB | `goobla run mistral`             |
| Moondream 2        | 1.4B       | 829MB | `goobla run moondream`           |
| Neural Chat        | 7B         | 4.1GB | `goobla run neural-chat`         |
| Starling           | 7B         | 4.1GB | `goobla run starling-lm`         |
| Code Llama         | 7B         | 3.8GB | `goobla run codellama`           |
| Llama 2 Uncensored | 7B         | 3.8GB | `goobla run llama2-uncensored`   |
| LLaVA              | 7B         | 4.5GB | `goobla run llava`               |
| Granite-3.3         | 8B         | 4.9GB | `goobla run granite3.3`          |

> [!NOTE]
> You should have at least 8 GB of RAM available to run the 7B models, 16 GB to run the 13B models, and 32 GB to run the 33B models.

## Customize a model

### Import from GGUF

Goobla supports importing GGUF models in the Modelfile:

1. Create a file named `Modelfile`, with a `FROM` instruction with the local filepath to the model you want to import.

   ```
   FROM ./vicuna-33b.Q4_0.gguf
   ```

2. Create the model in Goobla

   ```shell
   goobla create example -f Modelfile
   ```

3. Run the model

   ```shell
   goobla run example
   ```

### Import from Safetensors

See the [guide](docs/import.md) on importing models for more information.

### Customize a prompt

Models from the Goobla library can be customized with a prompt. For example, to customize the `llama3.2` model:

```shell
goobla pull llama3.2
```

Create a `Modelfile`:

```
FROM llama3.2

# set the temperature to 1 [higher is more creative, lower is more coherent]
PARAMETER temperature 1

# set the system message
SYSTEM """
You are Mario from Super Mario Bros. Answer as Mario, the assistant, only.
"""
```

Next, create and run the model:

```
goobla create mario -f ./Modelfile
goobla run mario
>>> hi
Hello! It's your friend Mario.
```

For more information on working with a Modelfile, see the [Modelfile](docs/modelfile.md) documentation.

## CLI Reference

### Create a model

`goobla create` is used to create a model from a Modelfile.

```shell
goobla create mymodel -f ./Modelfile
```

### Pull a model

```shell
goobla pull llama3.2
```

> This command can also be used to update a local model. Only the diff will be pulled.

### Remove a model

```shell
goobla rm llama3.2
```

### Copy a model

```shell
goobla cp llama3.2 my-model
```

### Multiline input

For multiline input, you can wrap text with `"""`:

```
>>> """Hello,
... world!
... """
I'm a basic program that prints the famous "Hello, world!" message to the console.
```

### Multimodal models

```
goobla run llava "What's in this image? /Users/jmorgan/Desktop/smile.png"
```

> **Output**: The image features a yellow smiley face, which is likely the central focus of the picture.

### Pass the prompt as an argument

```shell
goobla run llama3.2 "Summarize this file: $(cat README.md)"
```

> **Output**: Goobla is a lightweight, extensible framework for building and running language models on the local machine. It provides a simple API for creating, running, and managing models, as well as a library of pre-built models that can be easily used in a variety of applications.

### Show model information

```shell
goobla show llama3.2
```

### List models on your computer

```shell
goobla list
```

### List which models are currently loaded

```shell
goobla ps
```

### Stop a model which is currently running

```shell
goobla stop llama3.2
```

### Start Goobla

`goobla serve` is used when you want to start Goobla without running the desktop application.

## Building

See the [developer guide](https://github.com/goobla/goobla/blob/main/docs/development.md)

### Running local builds

Next, start the server:

```shell
./goobla serve
```

Finally, in a separate shell, run a model:

```shell
./goobla run llama3.2
```

## REST API

Goobla has a REST API for running and managing models.

### Generate a response

```shell
curl http://localhost:11434/api/generate -d '{
  "model": "llama3.2",
  "prompt":"Why is the sky blue?"
}'
```

### Chat with a model

```shell
curl http://localhost:11434/api/chat -d '{
  "model": "llama3.2",
  "messages": [
    { "role": "user", "content": "why is the sky blue?" }
  ]
}'
```

See the [API documentation](./docs/api.md) for all endpoints.

## Community Integrations

### Web & Desktop

- [Open WebUI](https://github.com/open-webui/open-webui)
- [SwiftChat (macOS with ReactNative)](https://github.com/aws-samples/swift-chat)
- [Enchanted (macOS native)](https://github.com/AugustDev/enchanted)
- [Hgoobla](https://github.com/fmaclen/hgoobla)
- [Lollms-Webui](https://github.com/ParisNeo/lollms-webui)
- [LibreChat](https://github.com/danny-avila/LibreChat)
- [Bionic GPT](https://github.com/bionic-gpt/bionic-gpt)
- [HTML UI](https://github.com/rtcfirefly/goobla-ui)
- [Saddle](https://github.com/jikkuatwork/saddle)
- [TagSpaces](https://www.tagspaces.org) (A platform for file-based apps, [utilizing Goobla](https://docs.tagspaces.org/ai/) for the generation of tags and descriptions)
- [Chatbot UI](https://github.com/ivanfioravanti/chatbot-goobla)
- [Chatbot UI v2](https://github.com/mckaywrigley/chatbot-ui)
- [Typescript UI](https://github.com/goobla-interface/Goobla-Gui?tab=readme-ov-file)
- [Minimalistic React UI for Goobla Models](https://github.com/richawo/minimal-llm-ui)
- [Gooblac](https://github.com/kevinhermawan/Gooblac)
- [big-AGI](https://github.com/enricoros/big-AGI)
- [Cheshire Cat assistant framework](https://github.com/cheshire-cat-ai/core)
- [Amica](https://github.com/semperai/amica)
- [chatd](https://github.com/BruceMacD/chatd)
- [Goobla-SwiftUI](https://github.com/kghandour/Goobla-SwiftUI)
- [Dify.AI](https://github.com/langgenius/dify)
- [MindMac](https://mindmac.app)
- [NextJS Web Interface for Goobla](https://github.com/jakobhoeg/nextjs-goobla-llm-ui)
- [Msty](https://msty.app)
- [Chatbox](https://github.com/Bin-Huang/Chatbox)
- [WinForm Goobla Copilot](https://github.com/tgraupmann/WinForm_Goobla_Copilot)
- [NextChat](https://github.com/ChatGPTNextWeb/ChatGPT-Next-Web) with [Get Started Doc](https://docs.nextchat.dev/models/goobla)
- [Alpaca WebUI](https://github.com/mmo80/alpaca-webui)
- [GooblaGUI](https://github.com/enoch1118/gooblaGUI)
- [OpenAOE](https://github.com/InternLM/OpenAOE)
- [Odin Runes](https://github.com/leonid20000/OdinRunes)
- [LLM-X](https://github.com/mrdjohnson/llm-x) (Progressive Web App)
- [AnythingLLM (Docker + MacOs/Windows/Linux native app)](https://github.com/Mintplex-Labs/anything-llm)
- [Goobla Basic Chat: Uses HyperDiv Reactive UI](https://github.com/rapidarchitect/goobla_basic_chat)
- [Goobla-chats RPG](https://github.com/drazdra/goobla-chats)
- [IntelliBar](https://intellibar.app/) (AI-powered assistant for macOS)
- [Jirapt](https://github.com/AliAhmedNada/jirapt) (Jira Integration to generate issues, tasks, epics)
- [ojira](https://github.com/AliAhmedNada/ojira) (Jira chrome plugin to easily generate descriptions for tasks)
- [QA-Pilot](https://github.com/reid41/QA-Pilot) (Interactive chat tool that can leverage Goobla models for rapid understanding and navigation of GitHub code repositories)
- [ChatGoobla](https://github.com/sugarforever/chat-goobla) (Open Source Chatbot based on Goobla with Knowledge Bases)
- [CRAG Goobla Chat](https://github.com/Nagi-ovo/CRAG-Goobla-Chat) (Simple Web Search with Corrective RAG)
- [RAGFlow](https://github.com/infiniflow/ragflow) (Open-source Retrieval-Augmented Generation engine based on deep document understanding)
- [StreamDeploy](https://github.com/StreamDeploy-DevRel/streamdeploy-llm-app-scaffold) (LLM Application Scaffold)
- [chat](https://github.com/swuecho/chat) (chat web app for teams)
- [Lobe Chat](https://github.com/lobehub/lobe-chat) with [Integrating Doc](https://lobehub.com/docs/self-hosting/examples/goobla)
- [Goobla RAG Chatbot](https://github.com/datvodinh/rag-chatbot.git) (Local Chat with multiple PDFs using Goobla and RAG)
- [BrainSoup](https://www.nurgo-software.com/products/brainsoup) (Flexible native client with RAG & multi-agent automation)
- [macai](https://github.com/Renset/macai) (macOS client for Goobla, ChatGPT, and other compatible API back-ends)
- [RWKV-Runner](https://github.com/josStorer/RWKV-Runner) (RWKV offline LLM deployment tool, also usable as a client for ChatGPT and Goobla)
- [Goobla Grid Search](https://github.com/dezoito/goobla-grid-search) (app to evaluate and compare models)
- [Olpaka](https://github.com/Otacon/olpaka) (User-friendly Flutter Web App for Goobla)
- [Casibase](https://casibase.org) (An open source AI knowledge base and dialogue system combining the latest RAG, SSO, goobla support, and multiple large language models.)
- [GooblaSpring](https://github.com/CrazyNeil/GooblaSpring) (Goobla Client for macOS)
- [LLocal.in](https://github.com/kartikm7/llocal) (Easy to use Electron Desktop Client for Goobla)
- [Shinkai Desktop](https://github.com/dcSpark/shinkai-apps) (Two click install Local AI using Goobla + Files + RAG)
- [AiLama](https://github.com/zeyoyt/ailama) (A Discord User App that allows you to interact with Goobla anywhere in Discord)
- [Goobla with Google Mesop](https://github.com/rapidarchitect/goobla_mesop/) (Mesop Chat Client implementation with Goobla)
- [R2R](https://github.com/SciPhi-AI/R2R) (Open-source RAG engine)
- [Goobla-Kis](https://github.com/elearningshow/goobla-kis) (A simple easy-to-use GUI with sample custom LLM for Drivers Education)
- [OpenGPA](https://opengpa.org) (Open-source offline-first Enterprise Agentic Application)
- [Painting Droid](https://github.com/mateuszmigas/painting-droid) (Painting app with AI integrations)
- [Kerlig AI](https://www.kerlig.com/) (AI writing assistant for macOS)
- [AI Studio](https://github.com/MindWorkAI/AI-Studio)
- [Sidellama](https://github.com/gyopak/sidellama) (browser-based LLM client)
- [LLMStack](https://github.com/trypromptly/LLMStack) (No-code multi-agent framework to build LLM agents and workflows)
- [BoltAI for Mac](https://boltai.com) (AI Chat Client for Mac)
- [Harbor](https://github.com/av/harbor) (Containerized LLM Toolkit with Goobla as default backend)
- [PyGPT](https://github.com/szczyglis-dev/py-gpt) (AI desktop assistant for Linux, Windows, and Mac)
- [Alpaca](https://github.com/Jeffser/Alpaca) (An Goobla client application for Linux and macOS made with GTK4 and Adwaita)
- [AutoGPT](https://github.com/Significant-Gravitas/AutoGPT/blob/master/docs/content/platform/goobla.md) (AutoGPT Goobla integration)
- [Go-CREW](https://www.jonathanhecl.com/go-crew/) (Powerful Offline RAG in Golang)
- [PartCAD](https://github.com/openvmp/partcad/) (CAD model generation with OpenSCAD and CadQuery)
- [Goobla4j Web UI](https://github.com/goobla4j/goobla4j-web-ui) - Java-based Web UI for Goobla built with Vaadin, Spring Boot, and Goobla4j
- [PyOllaMx](https://github.com/kspviswa/pyOllaMx) - macOS application capable of chatting with both Goobla and Apple MLX models.
- [Cline](https://github.com/cline/cline) - Formerly known as Claude Dev is a VSCode extension for multi-file/whole-repo coding
- [Cherry Studio](https://github.com/kangfenmao/cherry-studio) (Desktop client with Goobla support)
- [ConfiChat](https://github.com/1runeberg/confichat) (Lightweight, standalone, multi-platform, and privacy-focused LLM chat interface with optional encryption)
- [Archyve](https://github.com/nickthecook/archyve) (RAG-enabling document library)
- [crewAI with Mesop](https://github.com/rapidarchitect/goobla-crew-mesop) (Mesop Web Interface to run crewAI with Goobla)
- [Tkinter-based client](https://github.com/chyok/goobla-gui) (Python tkinter-based Client for Goobla)
- [LLMChat](https://github.com/trendy-design/llmchat) (Privacy focused, 100% local, intuitive all-in-one chat interface)
- [Local Multimodal AI Chat](https://github.com/Leon-Sander/Local-Multimodal-AI-Chat) (Goobla-based LLM Chat with support for multiple features, including PDF RAG, voice chat, image-based interactions, and integration with OpenAI.)
- [ARGO](https://github.com/xark-argo/argo) (Locally download and run Goobla and Huggingface models with RAG on Mac/Windows/Linux)
- [OrionChat](https://github.com/EliasPereirah/OrionChat) - OrionChat is a web interface for chatting with different AI providers
- [G1](https://github.com/bklieger-groq/g1) (Prototype of using prompting strategies to improve the LLM's reasoning through o1-like reasoning chains.)
- [Web management](https://github.com/lemonit-eric-mao/goobla-web-management) (Web management page)
- [Promptery](https://github.com/promptery/promptery) (desktop client for Goobla.)
- [Goobla App](https://github.com/JHubi1/goobla-app) (Modern and easy-to-use multi-platform client for Goobla)
- [chat-goobla](https://github.com/annilq/chat-goobla) (a React Native client for Goobla)
- [SpaceLlama](https://github.com/tcsenpai/spacellama) (Firefox and Chrome extension to quickly summarize web pages with goobla in a sidebar)
- [YouLama](https://github.com/tcsenpai/youlama) (Webapp to quickly summarize any YouTube video, supporting Invidious as well)
- [DualMind](https://github.com/tcsenpai/dualmind) (Experimental app allowing two models to talk to each other in the terminal or in a web interface)
- [gooblarama-matrix](https://github.com/h1ddenpr0cess20/gooblarama-matrix) (Goobla chatbot for the Matrix chat protocol)
- [goobla-chat-app](https://github.com/anan1213095357/goobla-chat-app) (Flutter-based chat app)
- [Perfect Memory AI](https://www.perfectmemory.ai/) (Productivity AI assists personalized by what you have seen on your screen, heard, and said in the meetings)
- [Hexabot](https://github.com/hexastack/hexabot) (A conversational AI builder)
- [Reddit Rate](https://github.com/rapidarchitect/reddit_analyzer) (Search and Rate Reddit topics with a weighted summation)
- [OpenTalkGpt](https://github.com/adarshM84/OpenTalkGpt) (Chrome Extension to manage open-source models supported by Goobla, create custom models, and chat with models from a user-friendly UI)
- [VT](https://github.com/vinhnx/vt.ai) (A minimal multimodal AI chat app, with dynamic conversation routing. Supports local models via Goobla)
- [Nosia](https://github.com/nosia-ai/nosia) (Easy to install and use RAG platform based on Goobla)
- [Witsy](https://github.com/nbonamy/witsy) (An AI Desktop application available for Mac/Windows/Linux)
- [Abbey](https://github.com/US-Artificial-Intelligence/abbey) (A configurable AI interface server with notebooks, document storage, and YouTube support)
- [Minima](https://github.com/dmayboroda/minima) (RAG with on-premises or fully local workflow)
- [aidful-goobla-model-delete](https://github.com/AidfulAI/aidful-goobla-model-delete) (User interface for simplified model cleanup)
- [Perplexica](https://github.com/ItzCrazyKns/Perplexica) (An AI-powered search engine & an open-source alternative to Perplexity AI)
- [Goobla Chat WebUI for Docker ](https://github.com/oslook/goobla-webui) (Support for local docker deployment, lightweight goobla webui)
- [AI Toolkit for Visual Studio Code](https://aka.ms/ai-tooklit/goobla-docs) (Microsoft-official VSCode extension to chat, test, evaluate models with Goobla support, and use them in your AI applications.)
- [MinimalNextGooblaChat](https://github.com/anilkay/MinimalNextGooblaChat) (Minimal Web UI for Chat and Model Control)
- [Chipper](https://github.com/TilmanGriesel/chipper) AI interface for tinkerers (Goobla, Haystack RAG, Python)
- [ChibiChat](https://github.com/CosmicEventHorizon/ChibiChat) (Kotlin-based Android app to chat with Goobla and Koboldcpp API endpoints)
- [LocalLLM](https://github.com/qusaismael/localllm) (Minimal Web-App to run goobla models on it with a GUI)
- [Gooblazing](https://github.com/buiducnhat/gooblazing) (Web extension to run Goobla models)
- [OpenDeepResearcher-via-searxng](https://github.com/benhaotang/OpenDeepResearcher-via-searxng) (A Deep Research equivalent endpoint with Goobla support for running locally)
- [AntSK](https://github.com/AIDotNet/AntSK) (Out-of-the-box & Adaptable RAG Chatbot)
- [MaxKB](https://github.com/1Panel-dev/MaxKB/) (Ready-to-use & flexible RAG Chatbot)
- [yla](https://github.com/danielekp/yla) (Web interface to freely interact with your customized models)
- [LangBot](https://github.com/RockChinQ/LangBot) (LLM-based instant messaging bots platform, with Agents, RAG features, supports multiple platforms)
- [1Panel](https://github.com/1Panel-dev/1Panel/) (Web-based Linux Server Management Tool)
- [AstrBot](https://github.com/Soulter/AstrBot/) (User-friendly LLM-based multi-platform chatbot with a WebUI, supporting RAG, LLM agents, and plugins integration)
- [Reins](https://github.com/ibrahimcetin/reins) (Easily tweak parameters, customize system prompts per chat, and enhance your AI experiments with reasoning model support.)
- [Flufy](https://github.com/Aharon-Bensadoun/Flufy) (A beautiful chat interface for interacting with Goobla's API. Built with React, TypeScript, and Material-UI.)
- [Ellama](https://github.com/zeozeozeo/ellama) (Friendly native app to chat with an Goobla instance)
- [screenpipe](https://github.com/mediar-ai/screenpipe) Build agents powered by your screen history
- [Ollamb](https://github.com/hengkysteen/ollamb) (Simple yet rich in features, cross-platform built with Flutter and designed for Goobla. Try the [web demo](https://hengkysteen.github.io/demo/ollamb/).)
- [Writeopia](https://github.com/Writeopia/Writeopia) (Text editor with integration with Goobla)
- [AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) (AI collaborative workspace with Goobla, cross-platform and self-hostable)
- [Lumina](https://github.com/cushydigit/lumina.git) (A lightweight, minimal React.js frontend for interacting with Goobla servers)
- [Tiny Notepad](https://pypi.org/project/tiny-notepad) (A lightweight, notepad-like interface to chat with goobla available on PyPI)
- [macLlama (macOS native)](https://github.com/hellotunamayo/macLlama) (A native macOS GUI application for interacting with Goobla models, featuring a chat interface.) 
- [GPTranslate](https://github.com/philberndt/GPTranslate) (A fast and lightweight, AI powered desktop translation application written with Rust and Tauri. Features real-time translation with OpenAI/Azure/Goobla.)
- [goobla launcher](https://github.com/NGC13009/goobla-launcher) (A launcher for Goobla, aiming to provide users with convenient functions such as goobla server launching, management, or configuration.)
- [ai-hub](https://github.com/Aj-Seven/ai-hub) (AI Hub supports multiple models via API keys and Chat support via Goobla API.)

### Cloud

- [Google Cloud](https://cloud.google.com/run/docs/tutorials/gpu-gemma2-with-goobla)
- [Fly.io](https://fly.io/docs/python/do-more/add-goobla/)
- [Koyeb](https://www.koyeb.com/deploy/goobla)

### Terminal

- [oterm](https://github.com/ggozad/oterm)
- [Ellama Emacs client](https://github.com/s-kostyaev/ellama)
- [Emacs client](https://github.com/zweifisch/goobla)
- [negoobla](https://github.com/paradoxical-dev/negoobla) UI client for interacting with models from within Neovim
- [gen.nvim](https://github.com/David-Kunz/gen.nvim)
- [goobla.nvim](https://github.com/nomnivore/goobla.nvim)
- [ollero.nvim](https://github.com/marco-souza/ollero.nvim)
- [goobla-chat.nvim](https://github.com/gerazov/goobla-chat.nvim)
- [ogpt.nvim](https://github.com/huynle/ogpt.nvim)
- [gptel Emacs client](https://github.com/karthink/gptel)
- [Oatmeal](https://github.com/dustinblackman/oatmeal)
- [cmdh](https://github.com/pgibler/cmdh)
- [ooo](https://github.com/npahlfer/ooo)
- [shell-pilot](https://github.com/reid41/shell-pilot)(Interact with models via pure shell scripts on Linux or macOS)
- [tenere](https://github.com/pythops/tenere)
- [llm-goobla](https://github.com/taketwo/llm-goobla) for [Datasette's LLM CLI](https://llm.datasette.io/en/stable/).
- [typechat-cli](https://github.com/anaisbetts/typechat-cli)
- [ShellOracle](https://github.com/djcopley/ShellOracle)
- [tlm](https://github.com/yusufcanb/tlm)
- [podman-goobla](https://github.com/ericcurtin/podman-goobla)
- [ggoobla](https://github.com/sammcj/ggoobla)
- [ParLlama](https://github.com/paulrobello/parllama)
- [Goobla eBook Summary](https://github.com/cognitivetech/goobla-ebook-summary/)
- [Goobla Mixture of Experts (MOE) in 50 lines of code](https://github.com/rapidarchitect/goobla_moe)
- [vim-intelligence-bridge](https://github.com/pepo-ec/vim-intelligence-bridge) Simple interaction of "Goobla" with the Vim editor
- [x-cmd goobla](https://x-cmd.com/mod/goobla)
- [bb7](https://github.com/drunkwcodes/bb7)
- [SwgooblaCLI](https://github.com/marcusziade/Swgoobla) bundled with the Swgoobla Swift package. [Demo](https://github.com/marcusziade/Swgoobla?tab=readme-ov-file#cli-usage)
- [aichat](https://github.com/sigoden/aichat) All-in-one LLM CLI tool featuring Shell Assistant, Chat-REPL, RAG, AI tools & agents, with access to OpenAI, Claude, Gemini, Goobla, Groq, and more.
- [PowershAI](https://github.com/rrg92/powershai) PowerShell module that brings AI to terminal on Windows, including support for Goobla
- [DeepShell](https://github.com/Abyss-c0re/deepshell) Your self-hosted AI assistant. Interactive Shell, Files and Folders analysis.
- [orbiton](https://github.com/xyproto/orbiton) Configuration-free text editor and IDE with support for tab completion with Goobla.
- [orca-cli](https://github.com/molbal/orca-cli) Goobla Registry CLI Application - Browse, pull, and download models from Goobla Registry in your terminal.
- [GGUF-to-Goobla](https://github.com/jonathanhecl/gguf-to-goobla) - Importing GGUF to Goobla made easy (multiplatform)
- [AWS-Strands-With-Goobla](https://github.com/rapidarchitect/goobla_strands) - AWS Strands Agents with Goobla Examples
- [goobla-multirun](https://github.com/attogram/goobla-multirun) - A bash shell script to run a single prompt against any or all of your locally installed goobla models, saving the output and performance statistics as easily navigable web pages. ([Demo](https://attogram.github.io/ai_test_zone/))

### Apple Vision Pro

- [SwiftChat](https://github.com/aws-samples/swift-chat) (Cross-platform AI chat app supporting Apple Vision Pro via "Designed for iPad")
- [Enchanted](https://github.com/AugustDev/enchanted)

### Database

- [pgai](https://github.com/timescale/pgai) - PostgreSQL as a vector database (Create and search embeddings from Goobla models using pgvector)
   - [Get started guide](https://github.com/timescale/pgai/blob/main/docs/vectorizer-quick-start.md)
- [MindsDB](https://github.com/mindsdb/mindsdb/blob/staging/mindsdb/integrations/handlers/goobla_handler/README.md) (Connects Goobla models with nearly 200 data platforms and apps)
- [chromem-go](https://github.com/philippgille/chromem-go/blob/v0.5.0/embed_goobla.go) with [example](https://github.com/philippgille/chromem-go/tree/v0.5.0/examples/rag-wikipedia-goobla)
- [Kangaroo](https://github.com/dbkangaroo/kangaroo) (AI-powered SQL client and admin tool for popular databases)

### Package managers

- [Pacman](https://archlinux.org/packages/extra/x86_64/goobla/)
- [Gentoo](https://github.com/gentoo/guru/tree/master/app-misc/goobla)
- [Homebrew](https://formulae.brew.sh/formula/goobla)
- [Helm Chart](https://artifacthub.io/packages/helm/goobla-helm/goobla)
- [Guix channel](https://codeberg.org/tusharhero/goobla-guix)
- [Nix package](https://search.nixos.org/packages?show=goobla&from=0&size=50&sort=relevance&type=packages&query=goobla)
- [Flox](https://flox.dev/blog/goobla-part-one)

### Libraries

- [LangChain](https://python.langchain.com/docs/integrations/chat/goobla/) and [LangChain.js](https://js.langchain.com/docs/integrations/chat/goobla/) with [example](https://js.langchain.com/docs/tutorials/local_rag/)
- [Firebase Genkit](https://firebase.google.com/docs/genkit/plugins/goobla)
- [crewAI](https://github.com/crewAIInc/crewAI)
- [Yacana](https://remembersoftwares.github.io/yacana/) (User-friendly multi-agent framework for brainstorming and executing predetermined flows with built-in tool integration)
- [Spring AI](https://github.com/spring-projects/spring-ai) with [reference](https://docs.spring.io/spring-ai/reference/api/chat/goobla-chat.html) and [example](https://github.com/tzolov/goobla-tools)
- [LangChainGo](https://github.com/tmc/langchaingo/) with [example](https://github.com/tmc/langchaingo/tree/main/examples/goobla.completion-example)
- [LangChain4j](https://github.com/langchain4j/langchain4j) with [example](https://github.com/langchain4j/langchain4j-examples/tree/main/goobla-examples/src/main/java)
- [LangChainRust](https://github.com/Abraxas-365/langchain-rust) with [example](https://github.com/Abraxas-365/langchain-rust/blob/main/examples/llm_goobla.rs)
- [LangChain for .NET](https://github.com/tryAGI/LangChain) with [example](https://github.com/tryAGI/LangChain/blob/main/examples/LangChain.Samples.OpenAI/Program.cs)
- [LLPhant](https://github.com/theodo-group/LLPhant?tab=readme-ov-file#goobla)
- [LlamaIndex](https://docs.llamaindex.ai/en/stable/examples/llm/goobla/) and [LlamaIndexTS](https://ts.llamaindex.ai/modules/llms/available_llms/goobla)
- [LiteLLM](https://github.com/BerriAI/litellm)
- [GooblaFarm for Go](https://github.com/presbrey/gooblafarm)
- [GooblaSharp for .NET](https://github.com/awaescher/GooblaSharp)
- [Goobla for Ruby](https://github.com/gbaptista/goobla-ai)
- [Goobla-rs for Rust](https://github.com/pepperoni21/goobla-rs)
- [Goobla-hpp for C++](https://github.com/jmont-dev/goobla-hpp)
- [Goobla4j for Java](https://github.com/goobla4j/goobla4j)
- [ModelFusion Typescript Library](https://modelfusion.dev/integration/model-provider/goobla)
- [GooblaKit for Swift](https://github.com/kevinhermawan/GooblaKit)
- [Goobla for Dart](https://github.com/breitburg/dart-goobla)
- [Goobla for Laravel](https://github.com/cloudstudio/goobla-laravel)
- [LangChainDart](https://github.com/davidmigloz/langchain_dart)
- [Semantic Kernel - Python](https://github.com/microsoft/semantic-kernel/tree/main/python/semantic_kernel/connectors/ai/goobla)
- [Haystack](https://github.com/deepset-ai/haystack-integrations/blob/main/integrations/goobla.md)
- [Elixir LangChain](https://github.com/brainlid/langchain)
- [Goobla for R - rgoobla](https://github.com/JBGruber/rgoobla)
- [Goobla for R - goobla-r](https://github.com/hauselin/goobla-r)
- [Goobla-ex for Elixir](https://github.com/lebrunel/goobla-ex)
- [Goobla Connector for SAP ABAP](https://github.com/b-tocs/abap_btocs_goobla)
- [Testcontainers](https://testcontainers.com/modules/goobla/)
- [Portkey](https://portkey.ai/docs/welcome/integration-guides/goobla)
- [PromptingTools.jl](https://github.com/svilupp/PromptingTools.jl) with an [example](https://svilupp.github.io/PromptingTools.jl/dev/examples/working_with_goobla)
- [LlamaScript](https://github.com/Project-Llama/llamascript)
- [llm-axe](https://github.com/emirsahin1/llm-axe) (Python Toolkit for Building LLM Powered Apps)
- [Gollm](https://docs.gollm.co/examples/goobla-example)
- [Ggoobla for Golang](https://github.com/jonathanhecl/ggoobla)
- [Gooblaclient for Golang](https://github.com/xyproto/gooblaclient)
- [High-level function abstraction in Go](https://gitlab.com/tozd/go/fun)
- [Goobla PHP](https://github.com/ArdaGnsrn/goobla-php)
- [Agents-Flex for Java](https://github.com/agents-flex/agents-flex) with [example](https://github.com/agents-flex/agents-flex/tree/main/agents-flex-llm/agents-flex-llm-goobla/src/test/java/com/agentsflex/llm/goobla)
- [Parakeet](https://github.com/parakeet-nest/parakeet) is a GoLang library, made to simplify the development of small generative AI applications with Goobla.
- [Haverscript](https://github.com/andygill/haverscript) with [examples](https://github.com/andygill/haverscript/tree/main/examples)
- [Goobla for Swift](https://github.com/mattt/goobla-swift)
- [Swgoobla for Swift](https://github.com/marcusziade/Swgoobla) with [DocC](https://marcusziade.github.io/Swgoobla/documentation/swgoobla/)
- [GoLamify](https://github.com/prasad89/golamify)
- [Goobla for Haskell](https://github.com/tusharad/goobla-haskell)
- [multi-llm-ts](https://github.com/nbonamy/multi-llm-ts) (A Typescript/JavaScript library allowing access to different LLM in a unified API)
- [LlmTornado](https://github.com/lofcz/llmtornado) (C# library providing a unified interface for major FOSS & Commercial inference APIs)
- [Goobla for Zig](https://github.com/dravenk/goobla-zig)
- [Abso](https://github.com/lunary-ai/abso) (OpenAI-compatible TypeScript SDK for any LLM provider)
- [Nichey](https://github.com/goodreasonai/nichey) is a Python package for generating custom wikis for your research topic
- [Goobla for D](https://github.com/kassane/goobla-d)
- [GooblaPlusPlus](https://github.com/HardCodeDev777/GooblaPlusPlus) (Very simple C++ library for Goobla)

### Mobile

- [SwiftChat](https://github.com/aws-samples/swift-chat) (Lightning-fast Cross-platform AI chat app with native UI for Android, iOS, and iPad)
- [Enchanted](https://github.com/AugustDev/enchanted)
- [Maid](https://github.com/Mobile-Artificial-Intelligence/maid)
- [Goobla App](https://github.com/JHubi1/goobla-app) (Modern and easy-to-use multi-platform client for Goobla)
- [ConfiChat](https://github.com/1runeberg/confichat) (Lightweight, standalone, multi-platform, and privacy-focused LLM chat interface with optional encryption)
- [Goobla Android Chat](https://github.com/sunshine0523/GooblaServer) (No need for Termux, start the Goobla service with one click on an Android device)
- [Reins](https://github.com/ibrahimcetin/reins) (Easily tweak parameters, customize system prompts per chat, and enhance your AI experiments with reasoning model support.)

### Extensions & Plugins

- [Raycast extension](https://github.com/MassimilianoPasquini97/raycast_goobla)
- [Discgoobla](https://github.com/mxyng/discgoobla) (Discord bot inside the Goobla discord channel)
- [Continue](https://github.com/continuedev/continue)
- [Vibe](https://github.com/thewh1teagle/vibe) (Transcribe and analyze meetings with Goobla)
- [Obsidian Goobla plugin](https://github.com/hinterdupfinger/obsidian-goobla)
- [Logseq Goobla plugin](https://github.com/omagdy7/goobla-logseq)
- [NotesGoobla](https://github.com/andersrex/notesgoobla) (Apple Notes Goobla plugin)
- [Dagger Chatbot](https://github.com/samalba/dagger-chatbot)
- [Discord AI Bot](https://github.com/mekb-turtle/discord-ai-bot)
- [Goobla Telegram Bot](https://github.com/ruecat/goobla-telegram)
- [Hass Goobla Conversation](https://github.com/ej52/hass-goobla-conversation)
- [Rivet plugin](https://github.com/abrenneke/rivet-plugin-goobla)
- [Obsidian BMO Chatbot plugin](https://github.com/longy2k/obsidian-bmo-chatbot)
- [Cliobot](https://github.com/herval/cliobot) (Telegram bot with Goobla support)
- [Copilot for Obsidian plugin](https://github.com/logancyang/obsidian-copilot)
- [Obsidian Local GPT plugin](https://github.com/pfrankov/obsidian-local-gpt)
- [Open Interpreter](https://docs.openinterpreter.com/language-model-setup/local-models/goobla)
- [Llama Coder](https://github.com/ex3ndr/llama-coder) (Copilot alternative using Goobla)
- [Goobla Copilot](https://github.com/bernardo-bruning/goobla-copilot) (Proxy that allows you to use Goobla as a copilot like GitHub Copilot)
- [twinny](https://github.com/rjmacarthy/twinny) (Copilot and Copilot chat alternative using Goobla)
- [Wingman-AI](https://github.com/RussellCanfield/wingman-ai) (Copilot code and chat alternative using Goobla and Hugging Face)
- [Page Assist](https://github.com/n4ze3m/page-assist) (Chrome Extension)
- [Plasmoid Goobla Control](https://github.com/imoize/plasmoid-gooblacontrol) (KDE Plasma extension that allows you to quickly manage/control Goobla model)
- [AI Telegram Bot](https://github.com/tusharhero/aitelegrambot) (Telegram bot using Goobla in backend)
- [AI ST Completion](https://github.com/yaroslavyaroslav/OpenAI-sublime-text) (Sublime Text 4 AI assistant plugin with Goobla support)
- [Discord-Goobla Chat Bot](https://github.com/kevinthedang/discord-goobla) (Generalized TypeScript Discord Bot w/ Tuning Documentation)
- [ChatGPTBox: All in one browser extension](https://github.com/josStorer/chatGPTBox) with [Integrating Tutorial](https://github.com/josStorer/chatGPTBox/issues/616#issuecomment-1975186467)
- [Discord AI chat/moderation bot](https://github.com/rapmd73/Companion) Chat/moderation bot written in python. Uses Goobla to create personalities.
- [Headless Goobla](https://github.com/nischalj10/headless-goobla) (Scripts to automatically install goobla client & models on any OS for apps that depend on goobla server)
- [Terraform AWS Goobla & Open WebUI](https://github.com/xuyangbocn/terraform-aws-self-host-llm) (A Terraform module to deploy on AWS a ready-to-use Goobla service, together with its front-end Open WebUI service.)
- [node-red-contrib-goobla](https://github.com/jakubburkiewicz/node-red-contrib-goobla)
- [Local AI Helper](https://github.com/ivostoykov/localAI) (Chrome and Firefox extensions that enable interactions with the active tab and customisable API endpoints. Includes secure storage for user prompts.)
- [vnc-lm](https://github.com/jake83741/vnc-lm) (Discord bot for messaging with LLMs through Goobla and LiteLLM. Seamlessly move between local and flagship models.)
- [LSP-AI](https://github.com/SilasMarvin/lsp-ai) (Open-source language server for AI-powered functionality)
- [QodeAssist](https://github.com/Palm1r/QodeAssist) (AI-powered coding assistant plugin for Qt Creator)
- [Obsidian Quiz Generator plugin](https://github.com/ECuiDev/obsidian-quiz-generator)
- [AI Summmary Helper plugin](https://github.com/philffm/ai-summary-helper)
- [TextCraft](https://github.com/suncloudsmoon/TextCraft) (Copilot in Word alternative using Goobla)
- [Alfred Goobla](https://github.com/zeitlings/alfred-goobla) (Alfred Workflow)
- [TextLLaMA](https://github.com/adarshM84/TextLLaMA) A Chrome Extension that helps you write emails, correct grammar, and translate into any language
- [Simple-Discord-AI](https://github.com/zyphixor/simple-discord-ai)
- [LLM Telegram Bot](https://github.com/innightwolfsleep/llm_telegram_bot) (telegram bot, primary for RP. Oobabooga-like buttons, [A1111](https://github.com/AUTOMATIC1111/stable-diffusion-webui) API integration e.t.c)
- [mcp-llm](https://github.com/sammcj/mcp-llm) (MCP Server to allow LLMs to call other LLMs)
- [SimpleGooblaUnity](https://github.com/HardCodeDev777/SimpleGooblaUnity) (Unity Engine extension for communicating with Goobla in a few lines of code. Also works at runtime)
- [UnityCodeLama](https://github.com/HardCodeDev777/UnityCodeLama) (Unity Edtior tool to analyze scripts via Goobla)

### Supported backends

- [llama.cpp](https://github.com/ggerganov/llama.cpp) project founded by Georgi Gerganov.

### Observability
- [Opik](https://www.comet.com/docs/opik/cookbook/goobla) is an open-source platform to debug, evaluate, and monitor your LLM applications, RAG systems, and agentic workflows with comprehensive tracing, automated evaluations, and production-ready dashboards. Opik supports native intergration to Goobla.
- [Lunary](https://lunary.ai/docs/integrations/goobla) is the leading open-source LLM observability platform. It provides a variety of enterprise-grade features such as real-time analytics, prompt templates management, PII masking, and comprehensive agent tracing.
- [OpenLIT](https://github.com/openlit/openlit) is an OpenTelemetry-native tool for monitoring Goobla Applications & GPUs using traces and metrics.
- [HoneyHive](https://docs.honeyhive.ai/integrations/goobla) is an AI observability and evaluation platform for AI agents. Use HoneyHive to evaluate agent performance, interrogate failures, and monitor quality in production.
- [Langfuse](https://langfuse.com/docs/integrations/goobla) is an open source LLM observability platform that enables teams to collaboratively monitor, evaluate and debug AI applications.
- [MLflow Tracing](https://mlflow.org/docs/latest/llms/tracing/index.html#automatic-tracing) is an open source LLM observability tool with a convenient API to log and visualize traces, making it easy to debug and evaluate GenAI applications.
