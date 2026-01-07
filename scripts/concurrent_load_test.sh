#!/bin/bash
# Concurrent Load Test for GET /validator endpoint
# Tests what happens when 100 users call the API simultaneously with different validator IDs

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
NUM_REQUESTS="${NUM_REQUESTS:-100}"
CHAIN="${CHAIN:-mainnet}"
RANGE="${RANGE:-all_time}"
OUTPUT_DIR="load_test_results"
TIMEOUT_SECONDS=60

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Concurrent Load Test - GET /validator ${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "API URL:       ${YELLOW}${API_URL}${NC}"
echo -e "Requests:      ${YELLOW}${NUM_REQUESTS}${NC}"
echo -e "Chain:         ${YELLOW}${CHAIN}${NC}"
echo -e "Range:         ${YELLOW}${RANGE}${NC}"
echo -e "Timeout:       ${YELLOW}${TIMEOUT_SECONDS}s${NC}"
echo ""

# Check if the server is running
echo -e "${BLUE}Checking if server is available...${NC}"
if ! curl -s -o /dev/null -w "%{http_code}" "${API_URL}/health" | grep -q "200"; then
    echo -e "${RED}Error: Server is not responding at ${API_URL}/health${NC}"
    echo "Make sure the server is running with: go run cmd/server/main.go"
    exit 1
fi
echo -e "${GREEN}Server is up!${NC}"
echo ""

# Create output directory
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Function to make a single request
make_request() {
    local request_id=$1
    local validator_id=$2
    local start_time=$(date +%s.%N)
    
    # Make the request and capture response
    local http_code
    local response_file="${OUTPUT_DIR}/response_${request_id}.json"
    local timing_file="${OUTPUT_DIR}/timing_${request_id}.txt"
    
    http_code=$(curl -s -o "${response_file}" -w "%{http_code}" \
        --max-time ${TIMEOUT_SECONDS} \
        "${API_URL}/validator?ids=${validator_id}&chain=${CHAIN}&range=${RANGE}" 2>/dev/null) || http_code="000"
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "${end_time} - ${start_time}" | bc)
    
    # Write timing info
    echo "${request_id},${validator_id},${http_code},${duration}" >> "${OUTPUT_DIR}/results.csv"
    
    # Print progress
    if [ "${http_code}" = "200" ]; then
        echo -e "  Request ${request_id}: ${GREEN}HTTP ${http_code}${NC} - ${duration}s (validator ${validator_id})"
    elif [ "${http_code}" = "000" ]; then
        echo -e "  Request ${request_id}: ${RED}TIMEOUT${NC} - ${duration}s (validator ${validator_id})"
    else
        echo -e "  Request ${request_id}: ${RED}HTTP ${http_code}${NC} - ${duration}s (validator ${validator_id})"
    fi
}

# Export function and variables for parallel execution
export -f make_request
export API_URL CHAIN RANGE OUTPUT_DIR TIMEOUT_SECONDS
export RED GREEN YELLOW BLUE NC

# Initialize results file
echo "request_id,validator_id,http_code,duration_seconds" > "${OUTPUT_DIR}/results.csv"

echo -e "${BLUE}Starting ${NUM_REQUESTS} concurrent requests...${NC}"
echo -e "${YELLOW}(Each request uses a different validator ID from 1 to ${NUM_REQUESTS})${NC}"
echo ""

# Record start time
overall_start=$(date +%s.%N)

# Launch all requests in parallel
# Each request gets a unique validator ID (1 to NUM_REQUESTS)
for i in $(seq 1 ${NUM_REQUESTS}); do
    make_request $i $i &
done

# Wait for all background jobs to complete
wait

# Record end time
overall_end=$(date +%s.%N)
overall_duration=$(echo "${overall_end} - ${overall_start}" | bc)

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Results Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Calculate statistics
total=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | wc -l)
successful=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 == 200 {count++} END {print count+0}')
failed=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 != 200 {count++} END {print count+0}')
timeouts=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 == "000" {count++} END {print count+0}')

# Calculate timing stats for successful requests
if [ "${successful}" -gt 0 ]; then
    avg_duration=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 == 200 {sum += $4; count++} END {if (count > 0) printf "%.3f", sum/count; else print "N/A"}')
    min_duration=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 == 200 {if (min == "" || $4 < min) min = $4} END {printf "%.3f", min}')
    max_duration=$(tail -n +2 "${OUTPUT_DIR}/results.csv" | awk -F',' '$3 == 200 {if (max == "" || $4 > max) max = $4} END {printf "%.3f", max}')
else
    avg_duration="N/A"
    min_duration="N/A"
    max_duration="N/A"
fi

echo -e "Total requests:     ${YELLOW}${total}${NC}"
echo -e "Successful (200):   ${GREEN}${successful}${NC}"
echo -e "Failed:             ${RED}${failed}${NC}"
echo -e "Timeouts:           ${RED}${timeouts}${NC}"
echo ""
echo -e "Total wall time:    ${YELLOW}${overall_duration}s${NC}"
echo ""

if [ "${successful}" -gt 0 ]; then
    echo -e "${BLUE}Timing for successful requests:${NC}"
    echo -e "  Min duration:     ${min_duration}s"
    echo -e "  Max duration:     ${max_duration}s"
    echo -e "  Avg duration:     ${avg_duration}s"
fi

echo ""
echo -e "Detailed results saved to: ${YELLOW}${OUTPUT_DIR}/results.csv${NC}"
echo ""

# Analysis
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Analysis${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ "${timeouts}" -gt 0 ]; then
    echo -e "${RED}⚠ ${timeouts} requests timed out!${NC}"
    echo "  This is expected behavior due to the global rate limiter."
    echo "  With 1 req/sec to Beaconcha and 3+ calls per request,"
    echo "  concurrent requests queue up and eventually timeout."
    echo ""
fi

if [ "${successful}" -gt 0 ] && [ "${max_duration}" != "N/A" ]; then
    # Check if there's significant variance in response times
    if (( $(echo "${max_duration} > 10" | bc -l) )); then
        echo -e "${YELLOW}⚠ Large variance in response times detected.${NC}"
        echo "  This indicates requests are being serialized by the rate limiter."
        echo ""
    fi
fi

echo "The global rate limiter (1 req/sec to Beaconcha) creates a bottleneck"
echo "when multiple users call the API concurrently. Each user request requires"
echo "3+ Beaconcha API calls, so ${NUM_REQUESTS} concurrent users could require"
echo "$((NUM_REQUESTS * 3))+ seconds to complete all requests."
echo ""
