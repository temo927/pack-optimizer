# Pack Optimizer - Comprehensive Test Scenarios

## 1. Basic Functionality Tests

### 1.1 Standard Pack Sizes (250, 500, 1000, 2000, 5000)
- **Test 1.1.1**: Amount = 1 → Expected: 1 x 250 (minimal items, minimal packs)
- **Test 1.1.2**: Amount = 250 → Expected: 1 x 250 (exact match)
- **Test 1.1.3**: Amount = 251 → Expected: 1 x 500 (minimal items, minimal packs)
- **Test 1.1.4**: Amount = 500 → Expected: 1 x 500 (exact match)
- **Test 1.1.5**: Amount = 501 → Expected: 1 x 500 + 1 x 250 (minimal items, minimal packs)
- **Test 1.1.6**: Amount = 750 → Expected: 1 x 500 + 1 x 250
- **Test 1.1.7**: Amount = 1000 → Expected: 1 x 1000 (exact match)
- **Test 1.1.8**: Amount = 12001 → Expected: 2 x 5000 + 1 x 2000 + 1 x 250
- **Test 1.1.9**: Amount = 10000 → Expected: 2 x 5000 (exact match)

### 1.2 Edge Case Pack Sizes (23, 31, 53)
- **Test 1.2.1**: Amount = 500,000 → Expected: {23: 2, 31: 7, 53: 9429}
- **Test 1.2.2**: Amount = 1 → Expected: 1 x 23 (smallest pack)
- **Test 1.2.3**: Amount = 23 → Expected: 1 x 23 (exact match)
- **Test 1.2.4**: Amount = 24 → Expected: 1 x 31 (minimal items)
- **Test 1.2.5**: Amount = 31 → Expected: 1 x 31 (exact match)
- **Test 1.2.6**: Amount = 53 → Expected: 1 x 53 (exact match)
- **Test 1.2.7**: Amount = 54 → Expected: 1 x 53 + 1 x 23 (or similar minimal combination)

## 2. Boundary Condition Tests

### 2.1 Very Small Amounts
- **Test 2.1.1**: Amount = 0 → Should reject (validation error)
- **Test 2.1.2**: Amount = 1 → Should use smallest pack size
- **Test 2.1.3**: Amount = 2 → Should use smallest pack size
- **Test 2.1.4**: Amount = smallest pack size - 1 → Should use smallest pack

### 2.2 Very Large Amounts
- **Test 2.2.1**: Amount = 1,000,000 → Should calculate correctly
- **Test 2.2.2**: Amount = 10,000,000 → Should calculate correctly (if within limits)
- **Test 2.2.3**: Amount = 100,000,000 → Should handle or reject based on limits
- **Test 2.2.4**: Amount = 500,000 (with edge case sizes) → Already tested in 1.2.1

### 2.3 Exact Pack Size Multiples
- **Test 2.3.1**: Amount = 2 x smallest pack → Should use 2 packs of smallest size
- **Test 2.3.2**: Amount = 3 x largest pack → Should use 3 packs of largest size
- **Test 2.3.3**: Amount = combination of exact multiples → Should use exact packs

## 3. Pack Size Management Tests

### 3.1 Add Pack Sizes
- **Test 3.1.1**: Add single pack size (e.g., 750) → Should appear in list
- **Test 3.1.2**: Add duplicate pack size → Should be rejected or deduplicated
- **Test 3.1.3**: Add very small size (e.g., 1) → Should be accepted
- **Test 3.1.4**: Add very large size (e.g., 10000) → Should be accepted
- **Test 3.1.5**: Add negative number → Should be rejected
- **Test 3.1.6**: Add zero → Should be rejected
- **Test 3.1.7**: Add non-numeric value → Should be rejected
- **Test 3.1.8**: Add multiple sizes in sequence → All should appear

### 3.2 Delete Pack Sizes
- **Test 3.2.1**: Delete single pack size → Should be removed from list
- **Test 3.2.2**: Delete all pack sizes → Should handle empty state
- **Test 3.2.3**: Delete and recalculate → Should use remaining sizes
- **Test 3.2.4**: Delete size used in previous calculation → Should still work

### 3.3 Pack Size Combinations
- **Test 3.3.1**: Single pack size available → All amounts use that size
- **Test 3.3.2**: Two pack sizes (e.g., 250, 500) → Should optimize correctly
- **Test 3.3.3**: Three pack sizes → Should optimize correctly
- **Test 3.3.4**: Many pack sizes (10+) → Should optimize correctly
- **Test 3.3.5**: Non-sequential sizes (e.g., 23, 31, 53) → Should optimize correctly

## 4. Optimization Rule Tests

### 4.1 Rule 1: Only Whole Packs
- **Test 4.1.1**: Verify no fractional packs in results
- **Test 4.1.2**: Verify all pack quantities are integers
- **Test 4.1.3**: Verify total items >= requested amount

### 4.2 Rule 2: Minimal Items (Precedence)
- **Test 4.2.1**: Amount = 251 with [250, 500] → Should choose 1 x 500 (not 2 x 250)
- **Test 4.2.2**: Amount = 501 with [250, 500, 1000] → Should choose 1 x 500 + 1 x 250 (not 1 x 1000)
- **Test 4.2.3**: Verify overage is minimized across all test cases

### 4.3 Rule 3: Minimal Packs (Tie-breaker)
- **Test 4.3.1**: When multiple solutions have same total items, choose fewer packs
- **Test 4.3.2**: Amount = 1000 with [500, 1000] → Should choose 1 x 1000 (not 2 x 500)
- **Test 4.3.3**: Verify pack count is minimized when total items are equal

## 5. API Endpoint Tests

### 5.1 GET /api/v1/packs
- **Test 5.1.1**: Get default pack sizes → Should return [250, 500, 1000, 2000, 5000]
- **Test 5.1.2**: Get after adding sizes → Should return updated list
- **Test 5.1.3**: Get after deleting sizes → Should return updated list
- **Test 5.1.4**: Response format → Should be JSON with "sizes" array

### 5.2 PUT /api/v1/packs
- **Test 5.2.1**: Replace with new sizes → Should update successfully
- **Test 5.2.2**: Replace with empty array → Should handle gracefully
- **Test 5.2.3**: Replace with invalid sizes → Should reject with error
- **Test 5.2.4**: Replace with duplicate sizes → Should deduplicate
- **Test 5.2.5**: Replace with unsorted sizes → Should sort automatically

### 5.3 DELETE /api/v1/packs/{size}
- **Test 5.3.1**: Delete existing size → Should remove successfully
- **Test 5.3.2**: Delete non-existent size → Should handle gracefully
- **Test 5.3.3**: Delete last size → Should handle empty state
- **Test 5.3.4**: Delete and verify in GET → Should not appear in list

### 5.4 POST /api/v1/calculate
- **Test 5.4.1**: Calculate with default sizes → Should return correct result
- **Test 5.4.2**: Calculate with custom sizes → Should use provided sizes
- **Test 5.4.3**: Calculate with invalid amount → Should reject with error
- **Test 5.4.4**: Calculate with negative amount → Should reject
- **Test 5.4.5**: Calculate with zero amount → Should reject
- **Test 5.4.6**: Calculate with very large amount → Should handle or reject
- **Test 5.4.7**: Response format → Should include amount, totalItems, overage, totalPacks, breakdown

### 5.5 GET /api/v1/healthz
- **Test 5.5.1**: Health check → Should return 200 OK
- **Test 5.5.2**: Health check when DB down → Should handle gracefully

## 6. UI/UX Tests

### 6.1 Pack Size Management UI
- **Test 6.1.1**: Display current pack sizes → Should show all sizes as chips
- **Test 6.1.2**: Add size input → Should only accept digits
- **Test 6.1.3**: Add size button → Should add and refresh list
- **Test 6.1.4**: Delete size button (×) → Should remove size
- **Test 6.1.5**: Empty state → Should handle when no sizes exist
- **Test 6.1.6**: Success message → Should show after successful add/delete
- **Test 6.1.7**: Error message → Should show on validation errors

### 6.2 Calculator UI
- **Test 6.2.1**: Amount input → Should only accept digits
- **Test 6.2.2**: Calculate button → Should trigger calculation
- **Test 6.2.3**: Results display → Should show breakdown table
- **Test 6.2.4**: Loading state → Should show during calculation
- **Test 6.2.5**: Error handling → Should display API errors
- **Test 6.2.6**: Empty results → Should handle gracefully

### 6.3 Responsive Design
- **Test 6.3.1**: Mobile viewport → Should be usable on small screens
- **Test 6.3.2**: Tablet viewport → Should layout correctly
- **Test 6.3.3**: Desktop viewport → Should use full width appropriately

## 7. Error Handling Tests

### 7.1 Input Validation
- **Test 7.1.1**: Negative pack size → Should reject
- **Test 7.1.2**: Zero pack size → Should reject
- **Test 7.1.3**: Non-numeric pack size → Should reject
- **Test 7.1.4**: Negative amount → Should reject
- **Test 7.1.5**: Zero amount → Should reject
- **Test 7.1.6**: Non-numeric amount → Should reject
- **Test 7.1.7**: Extremely large values → Should handle or reject appropriately

### 7.2 API Error Handling
- **Test 7.2.1**: Network error → Should display user-friendly message
- **Test 7.2.2**: 400 Bad Request → Should show validation error
- **Test 7.2.3**: 500 Server Error → Should show error message
- **Test 7.2.4**: Timeout → Should handle gracefully

### 7.3 Edge Cases
- **Test 7.3.1**: No pack sizes available → Should prevent calculation
- **Test 7.3.2**: All pack sizes deleted → Should handle empty state
- **Test 7.3.3**: Concurrent modifications → Should handle race conditions

## 8. Performance Tests

### 8.1 Calculation Performance
- **Test 8.1.1**: Small amount (1-1000) → Should be instant (<100ms)
- **Test 8.1.2**: Medium amount (1000-100000) → Should be fast (<500ms)
- **Test 8.1.3**: Large amount (100000-1000000) → Should complete (<2s)
- **Test 8.1.4**: Very large amount (500000) → Should complete reasonably

### 8.2 API Performance
- **Test 8.2.1**: GET /packs → Should be fast (<50ms)
- **Test 8.2.2**: PUT /packs → Should be fast (<100ms)
- **Test 8.2.3**: DELETE /packs/{size} → Should be fast (<100ms)
- **Test 8.2.4**: POST /calculate → Should scale with amount size

## 9. Integration Tests

### 9.1 Full Workflow
- **Test 9.1.1**: Add sizes → Calculate → Verify results → Delete sizes
- **Test 9.1.2**: Change sizes → Recalculate same amount → Verify different results
- **Test 9.1.3**: Multiple calculations in sequence → Should all work correctly

### 9.2 Data Persistence
- **Test 9.2.1**: Add sizes → Restart app → Sizes should persist
- **Test 9.2.2**: Delete sizes → Restart app → Deletions should persist
- **Test 9.2.3**: Multiple users → Should share same pack sizes (if applicable)

## 10. Special Scenarios

### 10.1 Prime Number Pack Sizes
- **Test 10.1.1**: Sizes = [7, 11, 13] → Test various amounts
- **Test 10.1.2**: Sizes = [23, 31, 53] → Already covered in 1.2

### 10.2 Sequential Pack Sizes
- **Test 10.2.1**: Sizes = [100, 200, 300, 400, 500] → Test optimization
- **Test 10.2.2**: Sizes = [1, 2, 3, 4, 5] → Test with small sizes

### 10.3 Large Pack Size Differences
- **Test 10.3.1**: Sizes = [1, 1000] → Test optimization
- **Test 10.3.2**: Sizes = [10, 10000] → Test optimization

### 10.4 Real-World Scenarios
- **Test 10.4.1**: Order of 263 items (from example) → Should return 1 x 500
- **Test 10.4.2**: Order of 12001 items (from example) → Should return 2 x 5000 + 1 x 2000 + 1 x 250
- **Test 10.4.3**: Common order amounts (100, 500, 1000, 5000) → Should optimize correctly

## 11. Caching Tests

### 11.1 Redis Cache
- **Test 11.1.1**: First calculation → Should cache result
- **Test 11.1.2**: Same calculation → Should use cache
- **Test 11.1.3**: Pack sizes changed → Cache should be invalidated
- **Test 11.1.4**: Cache expiration → Should refresh after TTL

## 12. Database Tests

### 12.1 Data Integrity
- **Test 12.1.1**: Add sizes → Verify in database
- **Test 12.1.2**: Delete sizes → Verify removal in database
- **Test 12.1.3**: Concurrent updates → Should handle correctly
- **Test 12.1.4**: Database connection loss → Should handle gracefully

## Test Execution Priority

### Critical (Must Pass)
- All tests in sections 1, 2, 4, 5.4, 7.1

### High Priority
- Tests in sections 3, 5 (except 5.4), 6, 7.2

### Medium Priority
- Tests in sections 8, 9, 10

### Low Priority
- Tests in sections 11, 12

## Automated Test Scripts

Consider creating automated tests for:
- Unit tests for calculator algorithm
- Integration tests for API endpoints
- E2E tests for UI workflows
- Performance benchmarks

