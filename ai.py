#!/usr/bin/env python3
"""
Test script for AvalaAI Translation API
Tests Persian-to-English translation for medical texts
"""

import requests
import json
import time

def test_avalai_translation():
    # Use your actual API key from .env file
    API_KEY = "aa-hmGFFZQ31CbqHvh4Ce4Zg7xbb584k3eNirvlqMppILA2wbqZ"
    BASE_URL = "https://api.avalai.ir/v1"
    
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # Test cases with Persian medical texts
    test_cases = [
        {
            "name": "Simple drug dose question",
            "persian": "دوز متوپرولول",
            "expected": "metoprolol dose"
        },
        {
            "name": "Maximum dose question",
            "persian": "حداکثر دوز وانکومایسین چقدر است؟",
            "expected": "What is the maximum dose of vancomycin?"
        },
        {
            "name": "Mixed content",
            "persian": "maximum dose وانکومایسین",
            "expected": "maximum dose vancomycin"
        },
        {
            "name": "Drug interaction",
            "persian": "تداخل دارویی آسپرین و وارفارین",
            "expected": "drug interaction aspirin and warfarin"
        }
    ]
    
    print("🧪 Testing AvalaAI Translation API")
    print("=" * 50)
    
    for i, test_case in enumerate(test_cases, 1):
        print(f"\nTest {i}: {test_case['name']}")
        print(f"Persian Input: {test_case['persian']}")
        
        # Prepare request payload
        payload = {
            "model": "gemini-2.5-flash-lite",
            "input": f"Translate this Persian medical text to clear English. Return only the English translation: {test_case['persian']}"
        }
        
        try:
            # Make API request
            print("📤 Sending request...")
            start_time = time.time()
            
            response = requests.post(
                f"{BASE_URL}/responses", 
                headers=headers, 
                json=payload,
                timeout=30
            )
            
            end_time = time.time()
            response_time = round((end_time - start_time) * 1000, 2)
            
            print(f"⏱️  Response time: {response_time}ms")
            print(f"📥 Status code: {response.status_code}")
            
            if response.status_code == 200:
                result = response.json()
                print("✅ Success!")
                print(f"Raw response: {json.dumps(result, indent=2)}")
                
                if "output_text" in result:
                    translation = result["output_text"].strip().strip('"')
                    print(f"🔤 Translation: '{translation}'")
                    print(f"📏 Length: {len(translation)} characters")
                    
                    if translation:
                        print("✅ Translation is not empty")
                    else:
                        print("❌ Translation is empty!")
                        
                else:
                    print("❌ No 'output_text' field in response")
                    
            else:
                print(f"❌ Error response:")
                try:
                    error_data = response.json()
                    print(json.dumps(error_data, indent=2))
                except:
                    print(response.text)
                    
        except requests.exceptions.Timeout:
            print("⏰ Request timed out (30s)")
        except requests.exceptions.RequestException as e:
            print(f"🌐 Network error: {e}")
        except Exception as e:
            print(f"💥 Unexpected error: {e}")
            
        print("-" * 50)
        
        # Add delay between requests
        if i < len(test_cases):
            time.sleep(1)
    
    print("\n🏁 Test completed!")

def test_api_connectivity():
    """Test basic API connectivity"""
    API_KEY = "aa-hmGFFZQ31CbqHvh4Ce4Zg7xbb584k3eNirvlqMppILA2wbqZ"
    BASE_URL = "https://api.avalai.ir/v1"
    
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # Simple test payload
    payload = {
        "model": "gemini-2.5-flash-lite",
        "input": "Hello, world!"
    }
    
    print("🔗 Testing API connectivity...")
    
    try:
        response = requests.post(
            f"{BASE_URL}/responses",
            headers=headers,
            json=payload,
            timeout=10
        )
        
        print(f"Status: {response.status_code}")
        if response.status_code == 401:
            print("❌ API Key is invalid or expired")
        elif response.status_code == 200:
            print("✅ API Key is valid")
        else:
            print(f"⚠️  Unexpected status code: {response.status_code}")
            
        print("Response:", response.json())
        
    except Exception as e:
        print(f"❌ Connection failed: {e}")

if __name__ == "__main__":
    print("🤖 AvalaAI Translation API Test")
    print("================================")
    
    # First test connectivity
    test_api_connectivity()
    print()
    
    # Then test translation
    test_avalai_translation()
