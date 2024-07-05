# Its looks like semver, but its not, because we cannot release patch on prev major.minor release
MAJOR_VERSION=1
MAJOR_LAST_COMMIT_HASH="d5a4dd1d9777df4dd771c6c646939455ab7f5976"

MINOR_LAST_COMMIT_HASH=$(git rev-list --invert-grep -i --grep='fix' ${MAJOR_LAST_COMMIT_HASH}..HEAD --no-merges -n 1)
MINOR_VERSION=$(git rev-list --invert-grep -i --grep='fix' ${MAJOR_LAST_COMMIT_HASH}..HEAD --no-merges --count)

[ "${MINOR_LAST_COMMIT_HASH}" = "" ] && MINOR_VERSION="0" 
[ "${MINOR_LAST_COMMIT_HASH}" = "" ] && MINOR_LAST_COMMIT_HASH=${MAJOR_LAST_COMMIT_HASH} 

PATCH_VERSION=$(git rev-list ${MINOR_LAST_COMMIT_HASH}..HEAD --no-merges --count)

APP_VERSION=v${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}

# get current commit hash for tag
COMMIT_HASH=$(git rev-parse HEAD)

# POST a new ref to repo via Github API
curl -s -X POST https://api.github.com/repos/${GITHUB_REPOSITORY}/git/refs \
	-H "Authorization: token ${GITHUB_TOKEN}" \
	-d @- << EOF
{
  "ref": "refs/tags/${APP_VERSION}",
  "sha": "${COMMIT_HASH}"
}
EOF
