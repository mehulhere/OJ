def twoSum(nums, target):
    seen = {}
    for i, num in enumerate(nums):
        complement = target - num
        if complement in seen:
            return [seen[complement], i]
        seen[num] = i
    return []

# Parse input in the expected format
# Input format: nums = [2,7,11,15], target = 9
input_str = input().strip()
parts = input_str.split(", target = ")
nums_str = parts[0].replace("nums = ", "")
nums = eval(nums_str)
target = int(parts[1])

# Call the solution function and print the result
result = twoSum(nums, target)
print(result) 
 