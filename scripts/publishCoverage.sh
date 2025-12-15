#!/bin/bash

#/******************************************************************************
 #Copyright 2021 IBM Corp.
 # Licensed under the Apache License, Version 2.0 (the "License");
 # you may not use this file except in compliance with the License.
 # You may obtain a copy of the License at
 #
 #     http://www.apache.org/licenses/LICENSE-2.0
 #
 # Unless required by applicable law or agreed to in writing, software
 # distributed under the License is distributed on an "AS IS" BASIS,
 # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 # See the License for the specific language governing permissions and
 # limitations under the License.
# *****************************************************************************/

set -euo pipefail
set +x

echo "===== Publishing the coverage results ====="

NEW_COVERAGE_SOURCE="$GITHUB_WORKSPACE/cover.html"

get_coverage() {
    local file="$1"
    if [[ -f "$file" ]]; then
        grep "%)" "$file" \
          | sed 's/[][()><%]/ /g' \
          | awk '{s+=$4}END{if(NR>0)printf "%.2f", s/NR; else print "0"}'
    else
        echo "0"
    fi
}

NEW_COVERAGE=$(get_coverage "$NEW_COVERAGE_SOURCE")

echo "Computed coverage: $NEW_COVERAGE%"

# Only comment on PRs
if [[ "$GITHUB_EVENT_NAME" == "pull_request" ]]; then
    PR_NUMBER=$(jq -r .pull_request.number "$GITHUB_EVENT_PATH")

    COMMENT_BODY="**Test Coverage:** ${NEW_COVERAGE}%"

    echo "Posting PR comment: $COMMENT_BODY"

    curl -s -X POST \
      -H "Authorization: token ${GHE_TOKEN}" \
      -H "Content-Type: application/json" \
      -d "{\"body\": \"$COMMENT_BODY\"}" \
      "https://api.github.com/repos/$GITHUB_REPOSITORY/issues/$PR_NUMBER/comments"
fi

echo "===== Coverage publishing finished ====="