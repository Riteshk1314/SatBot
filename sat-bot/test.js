const https = require('https');

// Test configuration
const TEST_CONFIG = {
  hostname: 'api.chatbot.saturnalia.in',
  port: 443,
  path: '/chat',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'User-Agent': 'Node.js Test Client'
  }
};

// Test data
const testMessages = [
  { message: "what is mirage" },
  { message: "hello" },
  { message: "how does AR work?" }
];

// Function to make HTTPS request
function makeRequest(data) {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify(data);
    
    const options = {
      ...TEST_CONFIG,
      headers: {
        ...TEST_CONFIG.headers,
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    console.log(`🚀 Testing with message: "${data.message}"`);
    console.log(`📡 Sending HTTPS request to: https://${options.hostname}${options.path}`);
    
    const startTime = Date.now();
    
    const req = https.request(options, (res) => {
      let responseBody = '';
      
      console.log(`📊 Status Code: ${res.statusCode}`);
      console.log(`📋 Headers:`, res.headers);
      
      res.on('data', (chunk) => {
        responseBody += chunk;
      });
      
      res.on('end', () => {
        const endTime = Date.now();
        const requestTime = (endTime - startTime) / 1000;
        
        try {
          const parsedResponse = JSON.parse(responseBody);
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            data: parsedResponse,
            requestTime: requestTime,
            rawResponse: responseBody
          });
        } catch (error) {
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            data: null,
            requestTime: requestTime,
            rawResponse: responseBody,
            parseError: error.message
          });
        }
      });
    });
    
    req.on('error', (error) => {
      const endTime = Date.now();
      const requestTime = (endTime - startTime) / 1000;
      
      reject({
        error: error.message,
        requestTime: requestTime
      });
    });
    
    // Set timeout
    req.setTimeout(30000, () => {
      req.destroy();
      reject({
        error: 'Request timeout (30s)',
        requestTime: 30
      });
    });
    
    req.write(postData);
    req.end();
  });
}

// Function to run a single test
async function runTest(testData, testNumber) {
  console.log(`\n${'='.repeat(60)}`);
  console.log(`🧪 TEST ${testNumber}: Testing API Route`);
  console.log(`${'='.repeat(60)}`);
  
  try {
    const result = await makeRequest(testData);
    
    console.log(`✅ Request completed successfully!`);
    console.log(`⏱️  Request Time: ${result.requestTime}s`);
    console.log(`📤 Sent Data:`, testData);
    console.log(`📥 Response:`, result.data);
    
    // Validate expected response structure
    if (result.data && result.data.response && result.data.response_time) {
      console.log(`✨ Response validation: PASSED`);
      console.log(`💬 Bot Response: "${result.data.response}"`);
      console.log(`⚡ Server Response Time: ${result.data.response_time}`);
    } else {
      console.log(`❌ Response validation: FAILED`);
      console.log(`🔍 Raw response:`, result.rawResponse);
    }
    
    return { success: true, result };
    
  } catch (error) {
    console.log(`❌ Request failed!`);
    console.log(`⏱️  Request Time: ${error.requestTime}s`);
    console.log(`🚨 Error:`, error.error);
    
    return { success: false, error };
  }
}

// Function to run all tests
async function runAllTests() {
  console.log(`🎯 Starting API Tests for https://api.chatbot.saturnalia.in/chat`);
  console.log(`📅 Test started at: ${new Date().toISOString()}`);
  
  const results = [];
  
  for (let i = 0; i < testMessages.length; i++) {
    const result = await runTest(testMessages[i], i + 1);
    results.push(result);
    
    // Wait 1 second between tests to be nice to the API
    if (i < testMessages.length - 1) {
      console.log(`⏳ Waiting 1 second before next test...`);
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }
  
  // Summary
  console.log(`\n${'='.repeat(60)}`);
  console.log(`📊 TEST SUMMARY`);
  console.log(`${'='.repeat(60)}`);
  
  const successful = results.filter(r => r.success).length;
  const failed = results.filter(r => !r.success).length;
  
  console.log(`✅ Successful tests: ${successful}/${results.length}`);
  console.log(`❌ Failed tests: ${failed}/${results.length}`);
  
  if (failed > 0) {
    console.log(`\n🚨 Failed test details:`);
    results.forEach((result, index) => {
      if (!result.success) {
        console.log(`   Test ${index + 1}: ${result.error.error}`);
      }
    });
  }
  
  console.log(`\n📅 Tests completed at: ${new Date().toISOString()}`);
}

// Function to run a quick single test
async function quickTest() {
  console.log(`🏃‍♂️ Running quick test...`);
  await runTest({ message: "what is mirage" }, 1);
}

// Main execution
if (require.main === module) {
  const args = process.argv.slice(2);
  
  if (args.includes('--quick')) {
    quickTest();
  } else {
    runAllTests();
  }
}

// Export functions for use in other files
module.exports = {
  makeRequest,
  runTest,
  runAllTests,
  quickTest
};