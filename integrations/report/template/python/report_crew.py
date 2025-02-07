# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import time

class ReportMetadata:
	"""
	ReportMetadata class to store report metadata
	"""
	name: str
	description: str
	path: str
	confluence_parent: str

	def __init__(self, name, description, path, confluence_parent=None):
		self.name = name
		self.description = description
		self.path = path
		self.confluence_parent = str(confluence_parent)

	@classmethod
	def from_dict(cls, data):
		return cls(
			name=data.get('name'),
			description=data.get('description'),
			path=data.get('path'),
			confluence_parent=data.get('confluence_parent')
		)

class Report:
	"""
	Report class to store report data
	"""
	metadata: ReportMetadata
	duration: float
	timestamp: str
	extra_data: dict

    # init with required fields
	def __init__(self, duration, timestamp, extra_data={}):
		self.duration = round(duration, 2)
		self.timestamp = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(timestamp))
		self.extra_data = extra_data

	# load metadata from file
	def load_metadata(self, path="report-metadata.json"):
		with open(path) as f:
			metadata = json.load(f)

		self.metadata = ReportMetadata.from_dict(metadata)

	# add data to report
	def add_data(self, data):
		self.extra_data.update(data)

	# export report to json file
	def export(self, path="report.json", custom_serializer=None):
		report = {
			"test_name": self.metadata.name,
			"test_description": self.metadata.description,
			"test_path": self.metadata.path,
			"test_duration": self.duration,
			"test_timestamp": self.timestamp,
		}

		if self.metadata.confluence_parent:
			report["test_confluence_parent_page_id"] = self.metadata.confluence_parent

		if self.extra_data:
			report.update(self.extra_data)

		with open(path, "w") as f:
			json.dump(report, f, indent=4, default=custom_serializer)
