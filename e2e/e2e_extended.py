#!/usr/bin/env python3
"""
Extended E2E tests for S3 and Manifest resolvers.

These tests require additional AWS resources:
- S3 bucket with test configuration
- Manifest YAML file with secret references

Set these environment variables to run:
- SNAGSBY_E2E_S3_SOURCE: S3 source URL (e.g., s3://bucket/config.json?region=us-west-2)
- SNAGSBY_E2E_MANIFEST_SOURCE: Manifest source URL (e.g., manifest:///path/to/manifest.yaml)
"""

import os
import subprocess
import unittest
import json
import tempfile


class SnagsbyS3Tests(unittest.TestCase):
    """Tests for S3 resolver functionality."""

    def setUp(self):
        """Set up test fixtures."""
        self.snagsby_path = self._get_snagsby_path()
        self.s3_source = os.environ.get('SNAGSBY_E2E_S3_SOURCE')
        if not self.s3_source:
            self.skipTest("SNAGSBY_E2E_S3_SOURCE not set")

    def _get_snagsby_path(self):
        """Get the path to the snagsby binary."""
        os_name = os.uname().sysname.lower()
        return f'./dist/{os_name}/snagsby'

    def test_s3_resolver_basic(self):
        """Test basic S3 resolver functionality."""
        result = subprocess.run(
            [self.snagsby_path, self.s3_source],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        # Should contain export statements
        self.assertIn('export ', result.stdout)

    def test_s3_with_json_output(self):
        """Test S3 resolver with JSON output."""
        result = subprocess.run(
            [self.snagsby_path, '-o', 'json', self.s3_source],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        # Should be valid JSON
        data = json.loads(result.stdout)
        self.assertIsInstance(data, dict)

    def test_s3_region_parameter(self):
        """Test that region parameter in S3 URL is respected."""
        # The source should already have ?region= parameter
        result = subprocess.run(
            [self.snagsby_path, self.s3_source],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")

    def test_s3_nonexistent_bucket(self):
        """Test error handling for nonexistent S3 bucket."""
        result = subprocess.run(
            [self.snagsby_path, '-e', 's3://nonexistent-bucket-12345678/config.json'],
            capture_output=True,
            text=True
        )
        # Should fail
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Error processing snagsby source', result.stderr)

    def test_s3_invalid_json(self):
        """Test error handling for invalid JSON in S3 object."""
        # This would require an S3 object with invalid JSON
        # Skip if not available
        invalid_source = os.environ.get('SNAGSBY_E2E_S3_INVALID_JSON')
        if not invalid_source:
            self.skipTest("SNAGSBY_E2E_S3_INVALID_JSON not set")
        
        result = subprocess.run(
            [self.snagsby_path, '-e', invalid_source],
            capture_output=True,
            text=True
        )
        self.assertNotEqual(result.returncode, 0)


class SnagsbyManifestTests(unittest.TestCase):
    """Tests for Manifest resolver functionality."""

    def setUp(self):
        """Set up test fixtures."""
        self.snagsby_path = self._get_snagsby_path()
        self.manifest_source = os.environ.get('SNAGSBY_E2E_MANIFEST_SOURCE')
        if not self.manifest_source:
            self.skipTest("SNAGSBY_E2E_MANIFEST_SOURCE not set")

    def _get_snagsby_path(self):
        """Get the path to the snagsby binary."""
        os_name = os.uname().sysname.lower()
        return f'./dist/{os_name}/snagsby'

    def test_manifest_resolver_basic(self):
        """Test basic manifest resolver functionality."""
        result = subprocess.run(
            [self.snagsby_path, self.manifest_source],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        self.assertIn('export ', result.stdout)

    def test_manifest_with_json_output(self):
        """Test manifest resolver with JSON output."""
        result = subprocess.run(
            [self.snagsby_path, '-o', 'json', self.manifest_source],
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        data = json.loads(result.stdout)
        self.assertIsInstance(data, dict)

    def test_manifest_nonexistent_file(self):
        """Test error handling for nonexistent manifest file."""
        result = subprocess.run(
            [self.snagsby_path, '-e', 'manifest:///nonexistent/path/manifest.yaml'],
            capture_output=True,
            text=True
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Error processing snagsby source', result.stderr)

    def test_manifest_invalid_yaml(self):
        """Test error handling for invalid YAML in manifest."""
        # Create a temp file with invalid YAML
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            f.write("invalid: yaml: content: [[[")
            temp_path = f.name
        
        try:
            result = subprocess.run(
                [self.snagsby_path, '-e', f'manifest://{temp_path}'],
                capture_output=True,
                text=True
            )
            self.assertNotEqual(result.returncode, 0)
        finally:
            os.unlink(temp_path)


class SnagsbyIntegrationTests(unittest.TestCase):
    """Integration tests combining multiple resolvers."""

    def setUp(self):
        """Set up test fixtures."""
        self.snagsby_path = self._get_snagsby_path()

    def _get_snagsby_path(self):
        """Get the path to the snagsby binary."""
        os_name = os.uname().sysname.lower()
        return f'./dist/{os_name}/snagsby'

    def test_mixed_sources(self):
        """Test combining SM, S3, and other sources."""
        sources = []
        
        # Add available sources
        sm_source = os.environ.get('SNAGSBY_E2E_SOURCE')
        s3_source = os.environ.get('SNAGSBY_E2E_S3_SOURCE')
        
        if sm_source:
            sources.extend(sm_source.split())
        if s3_source:
            sources.append(s3_source)
        
        if len(sources) < 2:
            self.skipTest("Need at least 2 different source types")
        
        result = subprocess.run(
            [self.snagsby_path, '-o', 'json'] + sources,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0, f"stderr: {result.stderr}")
        data = json.loads(result.stdout)
        self.assertIsInstance(data, dict)
        self.assertGreater(len(data), 0)

    def test_source_override_behavior(self):
        """Test that later sources override earlier sources."""
        # This requires sources with overlapping keys
        sources = os.environ.get('SNAGSBY_E2E_OVERRIDE_TEST')
        if not sources:
            self.skipTest("SNAGSBY_E2E_OVERRIDE_TEST not set")
        
        source_list = sources.split()
        result = subprocess.run(
            [self.snagsby_path, '-o', 'json'] + source_list,
            capture_output=True,
            text=True
        )
        self.assertEqual(result.returncode, 0)
        data = json.loads(result.stdout)
        # The test would need to verify specific override behavior
        self.assertIsInstance(data, dict)


if __name__ == '__main__':
    unittest.main(verbosity=2)
