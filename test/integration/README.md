# Integration Tests

This directory contains end-to-end integration tests for SMPC that require a real SIMPL Windows installation.

## Prerequisites

- SIMPL Windows must be installed at `C:\Program Files (x86)\Crestron\SIMPL\SIMPL Windows.exe`
- Tests must run with administrator privileges
- Tests require actual Windows environment (cannot run in CI/CD without Windows runners)

## Running Integration Tests

Integration tests are marked with the `integration` build tag and are **not** run by default.

### Run all integration tests

```powershell
go test -tags=integration ./test/integration -v
```

### Run specific integration test

```powershell
go test -tags=integration ./test/integration -v -run TestIntegration_SimpleCompile
```

### Run with timeout (recommended)

```powershell
go test -tags=integration ./test/integration -v -timeout 10m
```

## Test Fixtures

The `fixtures/` directory contains test `.smw` files used by integration tests:

- **simple.smw**: Minimal valid SIMPL Windows program for basic compilation tests
- **with-warnings.smw**: Program that compiles with warnings (optional - create if needed)
- **with-errors.smw**: Program with compilation errors (optional - create if needed)

## Writing New Integration Tests

1. Add the build tag at the top of your test file:

   ```go
   //go:build integration
   // +build integration
   ```

2. Check for admin privileges:

   ```go
   if !windows.IsElevated() {
       t.Skip("Integration tests require administrator privileges")
   }
   ```

3. Use the `compileFile()` helper function for E2E compilation tests

4. Always call the cleanup function:

   ```go
   result, cleanup := compileFile(t, fixturePath, false)
   defer cleanup()
   ```

## Test Coverage

Integration tests complement unit tests by:

- Testing actual SIMPL Windows interaction
- Validating dialog handling in real environment
- Testing full compilation workflow end-to-end
- Verifying Windows API calls work correctly

## Troubleshooting

### "Integration tests require administrator privileges"

- Run your terminal/IDE as Administrator
- Or use `RunAs` to elevate: `RunAs /user:Administrator powershell`

### "Fixture file should exist"

- Ensure you're running tests from the project root
- Check that fixtures exist in `test/integration/fixtures/`

### "SIMPL Windows should appear within timeout"

- Verify SIMPL Windows is installed at the expected path
- Increase timeout if your system is slow
- Check SIMPL Windows isn't already running

### Tests hang or don't cleanup

- Manually close SIMPL Windows
- Kill any stuck `smpwin.exe` processes
- Restart your test run

## CI/CD Considerations

Integration tests are **not suitable** for standard CI/CD pipelines because they:

- Require SIMPL Windows installation (proprietary software)
- Require Windows OS with GUI
- Require administrator privileges
- Take longer to execute (real UI interactions)

Consider running integration tests:

- Manually before releases
- On dedicated Windows test machines
- As part of nightly builds (if infrastructure supports it)
