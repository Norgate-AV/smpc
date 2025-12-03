#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

MISSING_SECRETS=()
MISSING_REPOS=()
MISSING_TOOLS=()
MISSING_AUTH=()
MISSING_PERMISSIONS=()
WARNINGS=()

print_status() {
    echo -e "${BLUE}$1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_secret() {
    local secret_name="$1"
    local description="$2"

    if [ -z "${!secret_name}" ]; then
        MISSING_SECRETS+=("$secret_name - $description")
        print_error "Missing: $secret_name"
    else
        print_success "Found: $secret_name"
    fi
}

check_github_repo() {
    local repo="$1"
    local description="$2"
    local token="$3"

    local http_status
    http_status=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: token ${token}" \
        "https://api.github.com/repos/$repo")

    if [ $http_status -eq 200 ]; then
        print_success "Repository exists: $repo"
    else
        MISSING_REPOS+=("$repo - $description (HTTP status: $http_status)")
        print_error "Missing repository: $repo (HTTP status: $http_status)"
    fi
}

check_tool() {
    local tool="$1"

    if ! command -v "$tool" &> /dev/null; then
        MISSING_TOOLS+=("$tool")
        print_error "Missing tool: $tool"
    else
        print_success "Found tool: $tool"
    fi
}

check_gh_auth() {
    if gh auth status &> /dev/null; then
        print_success "GitHub CLI is authenticated"
    else
        MISSING_AUTH+=("gh")
        print_error "GitHub CLI is not authenticated"
    fi
}

check_token_contents_permission() {
    local token="$1"
    local repo="$2"
    local token_name="$3"

    if [ -z "$token" ]; then
        return
    fi

    local sha
    sha=$(curl -s -H "Authorization: token $token" "https://api.github.com/repos/$repo/git/ref/heads/master" | jq -r '.object.sha')

    local branch_name="test-branch"

    # Create a new branch
    local http_status_create
    http_status_create=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST \
        -H "Authorization: token $token" \
        -d "{\"ref\":\"refs/heads/$branch_name\",\"sha\":\"$sha\"}" \
        "https://api.github.com/repos/$repo/git/refs")

    if [ "$http_status_create" -eq 201 ]; then
        # Delete the branch
        local http_status_delete
        http_status_delete=$(curl -s -o /dev/null -w "%{http_code}" \
            -X DELETE \
            -H "Authorization: token $token" \
            "https://api.github.com/repos/$repo/git/refs/heads/$branch_name")

        if [ "$http_status_delete" -eq 204 ]; then
            print_success "contents:write permission verified for $repo"
        else
            print_error "Failed to delete branch in $repo (HTTP status: $http_status_delete)"
            MISSING_PERMISSIONS+=("$token_name (failed to delete branch in $repo)")
        fi
    else
        print_error "Failed to create branch in $repo (HTTP status: $http_status_create)"
        MISSING_PERMISSIONS+=("$token_name (failed to create branch in $repo)")
    fi
}

check_goreleaser_env_vars() {
    print_status "Checking for referenced environment variables in .goreleaser.yml..."

    local env_vars
    env_vars=$(grep -oP '(?<={{ .Env.)\w+(?= }})' .goreleaser.yml | sort -u)

    for var in $env_vars; do
        if [ -z "${!var}" ]; then
            MISSING_SECRETS+=("$var - Referenced in .goreleaser.yml but not set")
            print_error "Missing: $var"
        fi
    done
}

# 1. Configuration validation
print_status "1. 🔍 Validating GoReleaser Configuration..."

if go tool goreleaser check 2>/dev/null; then
    print_success "Configuration is valid"
else
    echo "   Checking with warnings..."
    go tool goreleaser check || {
        if [[ $? -eq 2 ]]; then
            print_warning "Configuration valid but has deprecation warnings"
        else
            print_error "Configuration has errors"
            exit 1
        fi
    }
fi

check_goreleaser_env_vars
echo ""

# 2. Required secrets check
print_status "2. 📋 Checking Required Secrets..."

check_secret "GITHUB_TOKEN" "GitHub Actions token (auto-provided)"
check_secret "SCOOP_NORGATEAV_CRESTRON" "Token for Scoop repository"
echo ""

# 3. Required repositories check
print_status "3. 🏗️ Checking Required Repositories..."

check_github_repo "Norgate-AV/scoop-norgateav-crestron" "Scoop bucket repository" "${GITHUB_TOKEN}"
echo ""

# 4. Check build tools
print_status "4. 🔧 Checking Build Tools..."

check_tool "go"
check_tool "git"
check_tool "gh"
check_tool "git-cliff"
check_tool "typos"
check_tool "curl"
check_tool "jq"

if command -v go &> /dev/null; then
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    print_success "Go version: $GO_VERSION"
fi

echo ""

print_status "5. 🔐 Checking Authentication..."

if command -v gh &> /dev/null; then
    check_gh_auth
fi

print_status "6. 🔑 Checking Release Token Permissions..."

if [ -z "${SCOOP_NORGATEAV_CRESTRON}" ]; then
    print_warning "SCOOP_NORGATEAV_CRESTRON is not set; skipping permission checks"
    WARNINGS+=("SCOOP_NORGATEAV_CRESTRON is not set; skipping permission checks")
else
    REPOS_TO_CHECK=(
        "Norgate-AV/scoop-norgateav-crestron"
    )

    for repo in "${REPOS_TO_CHECK[@]}"; do
        check_token_contents_permission "${SCOOP_NORGATEAV_CRESTRON}" "$repo" "SCOOP_NORGATEAV_CRESTRON"
    done
fi
echo ""

print_status "7. 📊 Validation Summary"
echo "======================"

if [ ${#MISSING_SECRETS[@]} -eq 0 ] && \
    [ ${#MISSING_REPOS[@]} -eq 0 ] && \
    [ ${#MISSING_TOOLS[@]} -eq 0 ] && \
    [ ${#MISSING_AUTH[@]} -eq 0 ] && \
    [ ${#MISSING_PERMISSIONS[@]} -eq 0 ]; then
    print_success "🎉 All required prerequisites are in place!"

    if [ ${#WARNINGS[@]} -gt 0 ]; then
        echo -e "\n${YELLOW}⚠️  Optional items:${NC}"
        print_warning "Optional items:"
        for warning in "${WARNINGS[@]}"; do
            echo -e "${YELLOW}   • $warning${NC}"
        done
    fi
else
    print_error "Missing required prerequisites:"

    if [ ${#MISSING_SECRETS[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Missing Secrets:${NC}"
        for secret in "${MISSING_SECRETS[@]}"; do
            echo -e "${RED}   • $secret${NC}"
        done
    fi

    if [ ${#MISSING_REPOS[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Missing Repositories:${NC}"
        for repo in "${MISSING_REPOS[@]}"; do
            echo -e "${RED}   • $repo${NC}"
        done
    fi

    if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Missing Tools:${NC}"
        for tool in "${MISSING_TOOLS[@]}"; do
            echo -e "${RED}   • $tool${NC}"
        done
    fi

    if [ ${#MISSING_AUTH[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Missing Authentication:${NC}"
        for auth in "${MISSING_AUTH[@]}"; do
            echo -e "${RED}   • $auth${NC}"
        done
    fi

    if [ ${#MISSING_PERMISSIONS[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Missing Permissions:${NC}"
        for permission in "${MISSING_PERMISSIONS[@]}"; do
            echo -e "${RED}   • $permission${NC}"
        done
    fi

    exit 1
fi

# 8. GoReleaser Test (without publishing)
print_status "8. 🧪 Testing GoReleaser process (no publishing)..."


SKIP_FLAGS="--skip=publish"
GO_RELEASER_SUCCESS=false

if go tool goreleaser release --snapshot --clean $SKIP_FLAGS --verbose; then
    print_success "GoReleaser process completed successfully"
    GO_RELEASER_SUCCESS=true
else
    print_error "GoReleaser process failed"
fi
echo ""

# 9. Summary
print_status "9. 📋 Test Summary"
echo "==============="
echo ""

print_success "GoReleaser configuration is valid."
print_success "All required secrets are present."
print_success "All required repositories are accessible."
print_success "All required build tools are installed."
print_success "All release token permissions are verified."
print_success "GitHub CLI is authenticated."

if [ "$GO_RELEASER_SUCCESS" = true ]; then
    print_success "GoReleaser release process completed successfully."
else
    print_error "GoReleaser process failed."
    exit 1
fi

echo ""
print_success "🚀 Ready to release!"
