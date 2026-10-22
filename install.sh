#!/bin/sh
set -e

RESET="\\033[0m"
RED="\\033[31;1m"
GREEN="\\033[32;1m"
YELLOW="\\033[33;1m"
BLUE="\\033[34;1m"
WHITE="\\033[37;1m"

print_unsupported_platform()
{
    >&2 say_red "error: We're sorry, but it looks like Pulumi Customer Managed Workflow Agent is not supported on your platform"
    >&2 say_red "       We support 64-bit versions of Linux and macOS and are interested in supporting"
    >&2 say_red "       more platforms.  Please open an issue at https://github.com/pulumi/customer-managed-workflow-agent/issues/new/choose and"
    >&2 say_red "       let us know what platform you're using!"
}

say_green()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${GREEN}" "$1" "${RESET}"
    return 0
}

say_red()
{
    printf "%b%s%b\\n" "${RED}" "$1" "${RESET}"
}

say_yellow()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${YELLOW}" "$1" "${RESET}"
    return 0
}

say_blue()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${BLUE}" "$1" "${RESET}"
    return 0
}

say_white()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${WHITE}" "$1" "${RESET}"
    return 0
}

at_exit()
{
    EXIT_CODE=$?
    # Remove this run's temporary files whether or not the install succeeded.
    rm -f "${TARBALL_DEST}"
    rm -rf "${VERIFY_DIR}" "${EXTRACT_DIR}"
    if [ "${EXIT_CODE}" -ne 0 ]; then
        >&2 say_red
        >&2 say_red "We're sorry, but it looks like something might have gone wrong during installation."
        >&2 say_red "If you need help, please join us on https://slack.pulumi.com/"
    fi
}

TARBALL_DEST=""
VERIFY_DIR=""
EXTRACT_DIR=""
trap at_exit EXIT

VERSION=""
SILENT=""
SKIP_CHECKSUM=""
VERIFY_SIGNATURE=""
SOURCE=""
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            if [ "$2" != "latest" ]; then
                VERSION=$2
            fi
            ;;
        --silent)
            SILENT="--silent"
            ;;
        --skip-checksum)
            SKIP_CHECKSUM="true"
            ;;
        --verify-signature)
            VERIFY_SIGNATURE="true"
            ;;
        --source)
            if [ -z "$2" ]; then
                >&2 say_red "error: --source requires a value: github or cdn"
                exit 1
            fi
            SOURCE=$2
            ;;
     esac
     shift
done

case "${SOURCE}" in
    ""|github|cdn) ;;
    *)
        >&2 say_red "error: --source must be github or cdn, got '${SOURCE}'"
        exit 1
        ;;
esac

if [ -n "${VERIFY_SIGNATURE}" ]; then
    if [ -n "${SKIP_CHECKSUM}" ]; then
        >&2 say_red "error: --verify-signature verifies the checksums file, so it can't be combined with --skip-checksum"
        exit 1
    fi
    if ! command -v cosign >/dev/null; then
        >&2 say_red "error: --verify-signature requires cosign on your \$PATH: https://docs.sigstore.dev/cosign/system_config/installation/"
        exit 1
    fi
fi

if [ -z "${VERSION}" ]; then
    # Query pulumi.com/customer-managed-workflow-agent/latest-version for the most recent release. Because this approach
    # is now used by third parties as well (e.g., GitHub Actions virtual environments),
    # changes to this API should be made with care to avoid breaking any services that
    # rely on it (and ideally be accompanied by PRs to update them accordingly).

    if ! VERSION=$(curl --retry 3 --fail --silent -L "https://www.pulumi.com/customer-managed-workflow-agent/latest-version"); then
        >&2 say_red "error: could not determine latest version of Pulumi Customer Managed Workflow Agent, try passing --version X.Y.Z to"
        >&2 say_red "       install an explicit version"
        exit 1
    fi
fi

OS=""
case $(uname) in
    "Linux") OS="linux";;
    "Darwin") OS="darwin";;
    *)
        print_unsupported_platform
        exit 1
        ;;
esac

ARCH=""
case $(uname -m) in
    "x86_64") ARCH="x64";;
    "arm64") ARCH="arm64";;
    "aarch64") ARCH="arm64";;
    *)
        print_unsupported_platform
        exit 1
        ;;
esac

TARBALL_URL="https://github.com/pulumi/customer-managed-workflow-agent/releases/download/v${VERSION}/"
TARBALL_URL_FALLBACK="https://get.pulumi.com/releases/customer-managed-workflow-agent/"
TARBALL_PATH=customer-managed-workflow-agent-v${VERSION}-${OS}-${ARCH}.tar.gz

# Releases after v2.3.0 are signed keyless; earlier ones were signed with cosign.pub (see README.md).
if [ -n "${VERIFY_SIGNATURE}" ]; then
    MAJOR=$(echo "${VERSION}" | cut -d. -f1)
    MINOR=$(echo "${VERSION}" | cut -d. -f2)
    PATCH=$(echo "${VERSION}" | cut -d. -f3 | sed 's/[^0-9].*//')
    if [ "${MAJOR}" -lt 2 ] || { [ "${MAJOR}" -eq 2 ] && { [ "${MINOR}" -lt 3 ] || { [ "${MINOR}" -eq 3 ] && [ "${PATCH}" -eq 0 ]; }; }; }; then
        >&2 say_red "error: --verify-signature supports releases after v2.3.0. Verify v${VERSION} with cosign.pub as described in"
        >&2 say_red "       https://github.com/pulumi/customer-managed-workflow-agent/blob/main/README.md"
        exit 1
    fi
fi

if ! command -v customer-managed-workflow-agent >/dev/null; then
    say_blue "=== Installing Customer Managed Workflow Agent v${VERSION} ==="
else
    say_blue "=== Upgrading Customer Managed Workflow Agent $(customer-managed-workflow-agent version) to v${VERSION} ==="
fi

TARBALL_DEST=$(mktemp -t cmda.tar.gz.XXXXXXXXXX)

download_from_github() {
    say_white "+ Downloading ${TARBALL_URL}${TARBALL_PATH}..."
    # This should opportunistically use the GITHUB_TOKEN to avoid rate limiting
    # ...I think. It's hard to test accurately. But it at least doesn't seem to hurt.
    curl --fail ${SILENT} -L \
        --header "Authorization: Bearer $GITHUB_TOKEN" \
        -o "${TARBALL_DEST}" "${TARBALL_URL}${TARBALL_PATH}" || return 1
    CHECKSUMS_URL="${TARBALL_URL}checksums.txt"
}

download_from_cdn() {
    say_white "+ Downloading ${TARBALL_URL_FALLBACK}${TARBALL_PATH}..."
    curl --retry 2 --fail ${SILENT} -L -o "${TARBALL_DEST}" "${TARBALL_URL_FALLBACK}${TARBALL_PATH}" || return 1
    CHECKSUMS_URL="${TARBALL_URL_FALLBACK}customer-managed-workflow-agent-${VERSION}-checksums.txt"
}

download_tarball() {
    case "${SOURCE}" in
        github) download_from_github ;;
        cdn) download_from_cdn ;;
        *)
            # Try to download from github first, then fallback to get.pulumi.com
            if ! download_from_github; then
                say_white "+ Error encountered, falling back to get.pulumi.com..."
                download_from_cdn
            fi
            ;;
    esac
}

verify_tarball() {
    VERIFY_DIR=$(mktemp -dt cmda.verify.XXXXXXXXXX)
    say_white "+ Downloading ${CHECKSUMS_URL}..."
    if ! curl --retry 2 --fail ${SILENT} -L -o "${VERIFY_DIR}/checksums.txt" "${CHECKSUMS_URL}"; then
        >&2 say_red "error: failed to download ${CHECKSUMS_URL}"
        return 1
    fi

    if [ -n "${VERIFY_SIGNATURE}" ]; then
        say_white "+ Verifying the signature of the checksums file..."
        if ! curl --retry 2 --fail ${SILENT} -L -o "${VERIFY_DIR}/checksums.txt.sig" "${CHECKSUMS_URL}.sig"; then
            >&2 say_red "error: failed to download ${CHECKSUMS_URL}.sig"
            return 1
        fi
        if ! cosign verify-blob \
            --bundle "${VERIFY_DIR}/checksums.txt.sig" \
            --certificate-identity "https://github.com/pulumi/pulumi-service/.github/workflows/update-stack.yml@refs/heads/production" \
            --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
            "${VERIFY_DIR}/checksums.txt"; then
            >&2 say_red "error: the checksums file's signature did not verify"
            return 1
        fi
    fi

    say_white "+ Verifying the checksum of ${TARBALL_PATH}..."
    EXPECTED=$(awk -v f="${TARBALL_PATH}" '$2 == f || $2 == "*" f { print $1 }' "${VERIFY_DIR}/checksums.txt")
    if [ -z "${EXPECTED}" ]; then
        >&2 say_red "error: ${TARBALL_PATH} is not listed in ${CHECKSUMS_URL}"
        return 1
    fi
    if command -v sha256sum >/dev/null; then
        ACTUAL=$(sha256sum "${TARBALL_DEST}" | awk '{ print $1 }')
    elif command -v shasum >/dev/null; then
        ACTUAL=$(shasum -a 256 "${TARBALL_DEST}" | awk '{ print $1 }')
    else
        >&2 say_red "error: neither sha256sum nor shasum was found; pass --skip-checksum to install without checking"
        return 1
    fi
    if [ "${ACTUAL}" != "${EXPECTED}" ]; then
        >&2 say_red "error: checksum mismatch for ${TARBALL_PATH}: expected ${EXPECTED}, got ${ACTUAL}"
        return 1
    fi
}

if download_tarball; then
    if [ -z "${SKIP_CHECKSUM}" ]; then
        verify_tarball
    fi

    say_white "+ Extracting to $HOME/.pulumi/bin/customer-managed-workflow-agent"

    # If `~/.pulumi/bin/customer-managed-workflow-agent exists`, remove the existing files.
    # Note: handle files explicitly to avoid removing any existing configuration file
    if [ -e "${HOME}/.pulumi/bin/customer-managed-workflow-agent" ]; then
        rm -f "${HOME}/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent"
        rm -f "${HOME}"/.pulumi/bin/customer-managed-workflow-agent/workflow-runner*
        rm -f "${HOME}/.pulumi/bin/customer-managed-workflow-agent/pulumi-workflow-agent.yaml.sample"
    fi

    mkdir -p "${HOME}/.pulumi/bin/customer-managed-workflow-agent"

    # Yarn's shell installer does a similar dance of extracting to a temp
    # folder and copying to not depend on additional tar flags
    EXTRACT_DIR=$(mktemp -dt cmda.XXXXXXXXXX)
    tar zxf "${TARBALL_DEST}" -C "${EXTRACT_DIR}"

    cp "${EXTRACT_DIR}"/customer-managed-workflow-agent/* "${HOME}/.pulumi/bin/customer-managed-workflow-agent/"
else
    >&2 say_red "error: failed to download ${TARBALL_URL}"
    >&2 say_red "       check your internet and try again; if the problem persists, file an"
    >&2 say_red "       issue at https://github.com/pulumi/customer-managed-workflow-agent/issues/new/choose"
    exit 1
fi

# Now that we have installed the Customer Managed Workflow Agent, if it is not already on the path, let's add a line
# to the user's profile to add the folder to the PATH for future sessions.
if ! command -v customer-managed-workflow-agent >/dev/null; then
    # If we can, we'll add a line to the user's .profile adding $HOME/.pulumi/bin/customer-managed-workflow-agent to the PATH
    SHELL_NAME=$(basename "${SHELL}")
    PROFILE_FILE=""

    case "${SHELL_NAME}" in
        "bash")
            # Terminal.app on macOS prefers .bash_profile to .bashrc, so we prefer that
            # file when trying to put our export into a profile. On *NIX, .bashrc is
            # preferred as it is sourced for new interactive shells.
            if [ "$(uname)" != "Darwin" ]; then
                if [ -e "${HOME}/.bashrc" ]; then
                    PROFILE_FILE="${HOME}/.bashrc"
                elif [ -e "${HOME}/.bash_profile" ]; then
                    PROFILE_FILE="${HOME}/.bash_profile"
                fi
            else
                if [ -e "${HOME}/.bash_profile" ]; then
                    PROFILE_FILE="${HOME}/.bash_profile"
                elif [ -e "${HOME}/.bashrc" ]; then
                    PROFILE_FILE="${HOME}/.bashrc"
                fi
            fi
            ;;
        "zsh")
            if [ -e "${ZDOTDIR:-$HOME}/.zshrc" ]; then
                PROFILE_FILE="${ZDOTDIR:-$HOME}/.zshrc"
            fi
            ;;
    esac

    if [ -n "${PROFILE_FILE}" ]; then
        LINE_TO_ADD="export PATH=\$PATH:\$HOME/.pulumi/bin/customer-managed-workflow-agent"
        if ! grep -q "# add Pulumi Customer Managed Workflow Agent to the PATH" "${PROFILE_FILE}"; then
            say_white "+ Adding \$HOME/.pulumi/bin/customer-managed-workflow-agent to \$PATH in ${PROFILE_FILE}"
            printf "\\n# add Pulumi Customer Managed Workflow Agent to the PATH\\n%s\\n" "${LINE_TO_ADD}" >> "${PROFILE_FILE}"
        fi

        EXTRA_INSTALL_STEP="+ Please restart your shell or add $HOME/.pulumi/bin/customer-managed-workflow-agent to your \$PATH"
    else
        EXTRA_INSTALL_STEP="+ Please add $HOME/.pulumi/bin/customer-managed-workflow-agent to your \$PATH"
    fi
elif [ "$(command -v customer-managed-workflow-agent)" != "$HOME/.pulumi/bin/customer-managed-workflow-agent/customer-managed-workflow-agent" ]; then
    say_yellow
    say_yellow "warning: Pulumi Customer Managed Workflow Agent has been installed to $HOME/.pulumi/bin/customer-managed-workflow-agent, but it looks like there's a different copy"
    say_yellow "         on your \$PATH at $(dirname "$(command -v customer-managed-workflow-agent)"). You'll need to explicitly invoke the"
    say_yellow "         version you just installed or modify your \$PATH to prefer this location."
fi

check_for_docker() {
    say_blue
    say_blue "=== Checking for docker on \$PATH ==="

    if ! command -v docker >/dev/null; then
        say_red "+ docker not found. customer-managed-workflow-agent requires docker to be available on your \$PATH."
    else
        say_white "+ Confirmed"
    fi
}

check_for_docker

say_blue
say_blue "=== Pulumi Customer Managed Workflow Agent is now installed! ==="
if [ "$EXTRA_INSTALL_STEP" != "" ]; then
    say_white "${EXTRA_INSTALL_STEP}"
fi
say_green "+ Get started with Pulumi Customer Managed Workflow Agent: https://github.com/pulumi/customer-managed-workflow-agent/blob/main/README.md"
