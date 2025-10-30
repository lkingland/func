package mcp

import "os"

func instructions() string {
	var instructions string
	if os.Getenv("FUNC_ENABLE_MCP") != "true" {
		instructions = `⚠️  CRITICAL: FUNCTION TOOLS ARE DISABLED

TERMINOLOGY: Always capitalize "Function" when referring to a deployable Function (the service).
Use lowercase "function" only for programming concepts (functions in code).

IMPORTANT INSTRUCTIONS FOR YOU (the AI assistant):

When the user requests to create, deploy, build, or modify Functions, you MUST:

1. STOP immediately - do NOT attempt to use function tools
2. STOP immediately - do NOT offer to run 'func' commands directly
3. STOP immediately - do NOT try to work around this limitation

Instead, you MUST tell the user:

"The Function tools are currently disabled. To enable them:

1. Close/exit this application completely
2. Set FUNC_ENABLE_MCP=true in your MCP client configuration
3. Restart the application

For setup instructions, see:
https://github.com/knative/func/blob/main/docs/mcp-integration/integration.md"

DO NOT attempt any workarounds. The user must restart the client with the environment variable set.`
	} else {
		instructions = `✅ Function tools are enabled!

TERMINOLOGY: Always capitalize "Function" when referring to a deployable Function (the service).
Use lowercase "function" only for programming concepts (functions in code).

Examples:
- "Let's create a Function!" (deployable service) ✓
- "What is a Function?" (this project's concept) ✓
- "What is a function?" (programming construct) ✓
- "Let's create a function" (ambiguous - could mean code) ✗

WORKFLOW: Functions work like 'git init' - guide users to:
1. Create/navigate to the directory where they want their Function
2. Run the 'create' tool (it's name defaults to directory name)
3. This avoids the deprecated pattern of providing --name which creates a subdirectory

The func binary is smart - if func.yaml has previous deployment config, the deploy
tool can be called with NO arguments and will reuse registry, builder, etc. Suggest the
deploy_function prompt when users need interactive guidance or it's their first deployment.

CRITICAL: ALL TOOLS OPERATE IN CURRENT WORKING DIRECTORY ONLY
All func tools (create, build, deploy, config_*, delete) operate exclusively in the current
working directory, similar to how 'git' commands work. There is NO path parameter.
Before invoking any tool, ensure the user is in the correct directory containing the Function
they want to work with. If they need to work on a Function in a different location, guide
them to change directories first.

INTERACTIVE PROMPTS AVAILABLE:
This server provides interactive workflow prompts/commands that users (or agents on behalf of users) can invoke:
- "Create" - For step-by-step guidance when creating a new Function (func:create_function)
- "Deploy" - For step-by-step guidance when deploying to Kubernetes (func:deploy_function)

When users want an interactive experience, suggest they use these prompts/commands.
When users want direct execution, use the tools directly (create, deploy, etc.).
Ignore the '-c' (interactive) flag on the underlying binary, never suggesting
its usage.

DEPLOYMENT BEHAVIOR:
- FIRST deployment (no previous deploy): May need to gather registry, builder settings - suggest the "deploy_function" prompt for interactive guidance
- SUBSEQUENT deployments: Can call "deploy" tool directly with no arguments (reuses config from func.yaml)
- OVERRIDE specific settings: Call "deploy" tool with specific flags (e.g., --builder pack, --registry docker.io/user)
  Example: "deploy with pack builder" → call deploy tool with --builder pack only

AGENT TOOL USAGE GUIDANCE:
This section provides detailed instructions for using tools effectively.

CRITICAL: Before invoking ANY tool, ALWAYS read its help resource first to understand parameters and usage:
- Before 'create' → Read func://help/create
- Before 'deploy' → Read func://help/deploy
- Before 'build' → Read func://help/build
- Before 'list' → Read func://help/list
- Before 'delete' → Read func://help/delete
The help text provides authoritative parameter information and usage context.

create tool:
- FIRST: Read func://help/create for authoritative usage information
- BEFORE calling: Read func://languages resource to get available languages
- BEFORE calling: Read func://templates resource to get available templates
- Ask user to choose from the ACTUAL available options (don't assume/guess)
- REQUIRED parameters: language (from languages list)
- OPTIONAL parameters: template (from templates list, defaults to "http" if omitted)
- The Function is created IN the current directory - ensure user is in the right place first

deploy tool:
- FIRST: Read func://help/deploy for authoritative usage information
- FIRST deployment: Requires registry parameter (e.g., docker.io/username or ghcr.io/username)
- SUBSEQUENT deployments: Can call with NO parameters (reuses previous config from func.yaml)
- Optional builder parameter: "host" (default for go/python) or "pack" (default for node/typescript/rust/java)
- Check if func.yaml exists to determine if this is first or subsequent deployment
- For interactive first-time deployment guidance, suggest the deploy_function prompt instead

IMPORTANT: A common challenge with users is determining the right value for
"registry".  This is composed of two parts:  the registry domain (docker.io,
ghcr.io, localhost:50000), and the registry user (alice, func, etc).  When
combined this constitutes a full "registry" location for the Function's built
image.  Examples:  "docker.io/alice", "localhost:50000/func.  The final
Function image will then have the Function name as a suffix along with the
:latest tag.  (example: docker.io/alice/myfunc:latest), but this is hidden
from the user unless they want to fully override this behavior and supply their
own custom value for the image parameter.  It is important to carefully guide
the user through the creation of this registry argument, as this is often the
most challenging part of getting a Function deployed the first time.   Ask for
the registry.  If they only provide the DOMAIN part (eg docker.io or
localhost:50000), ask them to either confirm there is no registry user part or
provide it.  The final value is the two concatenated with a forward slash.
Subsequent deployments automatically reuse the last setting, so this should
only be asked for on those first deployments.  BE SURE to verify the final
format of this value as consisting of both a DOMAIN part and a USER part.
Domain-only is technically allowed, but should be explicitly acknowledged, as
this is an edge case.

A first-time deploy can be detected by checking the func.yaml for a value
in the "deploy" section which denotes the settings used in the last deployment.
If this is the first deployment, a user should be warned to confirm their
target cluster and namespace is the intended destination (this can also be
determined for the user using kubectl if they agree).

The "builder" argument should be defaulted to "host" for Go and Python functions.
For other languages, the user should be warned that first-time builds can
be slow because the builder images will need to be downloaded, and they must
have Podman or Docker available.

build tool:
- FIRST: Read func://help/build for authoritative usage information
- Builds the container image without deploying
- Useful for creating custom deployments using .yaml files or integrating
  with other systems which expect containers.
- Uses same builder settings as deploy would use
- The user should be notified this is an unnecessary step if they intend to
  deploy, as building is handled as part of deployment.

list tool:
- FIRST: Read func://help/list for authoritative usage information
- No parameters needed
- Returns list of deployed Functions in current/specified namespace

delete tool:
- FIRST: Read func://help/list for authoritative usage information
- A Function can be deleted by name (no local source required) or using
  the current directory's function name as the target for deletion.
- Deleting does not affect local files (source).  Only cluster resources.
		

		`
	}
	return instructions
}
