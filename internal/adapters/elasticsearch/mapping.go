package elasticsearch

// TalkPrivateIndexMapping defines the Elasticsearch mapping for the private talks index.
// This mapping includes all fields, including sensitive data like program committee
// feedback, submitter emails, and internal notes.
const TalkPrivateIndexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "default": {
          "type": "standard"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": {
        "type": "keyword"
      },
      "conferenceId": {
        "type": "keyword"
      },
      "conferenceSlug": {
        "type": "keyword"
      },
      "conferenceName": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword",
            "ignore_above": 256
          }
        }
      },
      "status": {
        "type": "keyword"
      },
      "lastUpdated": {
        "type": "date",
        "format": "strict_date_optional_time||epoch_millis"
      },
      "data": {
        "properties": {
          "title": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "abstract": {
            "type": "text"
          },
          "outline": {
            "type": "text"
          },
          "intendedAudience": {
            "type": "text"
          },
          "format": {
            "type": "keyword"
          },
          "language": {
            "type": "keyword"
          },
          "length": {
            "type": "keyword"
          },
          "level": {
            "type": "keyword"
          },
          "keywords": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "suggestedKeywords": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "suggestedCategory": {
            "type": "keyword"
          },
          "equipment": {
            "type": "text"
          },
          "infoToProgramCommittee": {
            "type": "text"
          },
          "participation": {
            "type": "text"
          },
          "postedBy": {
            "type": "keyword"
          },
          "room": {
            "type": "keyword"
          },
          "startTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "endTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "boardingTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "communicatedRoom": {
            "type": "keyword"
          },
          "communicatedStartTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "video": {
            "type": "keyword",
            "index": false
          },
          "slug": {
            "type": "keyword"
          },
          "published": {
            "type": "keyword"
          },
          "status": {
            "type": "keyword"
          },
          "workshopPrerequisites": {
            "type": "text"
          },
          "preparations": {
            "type": "text"
          },
          "tags": {
            "type": "keyword"
          },
          "tagswithauthor": {
            "type": "nested",
            "properties": {
              "author": {
                "type": "keyword"
              },
              "tag": {
                "type": "keyword"
              }
            }
          },
          "pkomfeedbacks": {
            "type": "nested",
            "properties": {
              "id": {
                "type": "keyword"
              },
              "talkid": {
                "type": "keyword"
              },
              "author": {
                "type": "keyword"
              },
              "feedbacktype": {
                "type": "keyword"
              },
              "info": {
                "type": "text"
              },
              "created": {
                "type": "keyword"
              }
            }
          },
          "feedback": {
            "properties": {
              "count": {
                "type": "integer"
              },
              "enjoySum": {
                "type": "integer"
              },
              "usefulSum": {
                "type": "integer"
              },
              "commentList": {
                "type": "text"
              }
            }
          }
        }
      },
      "speakers": {
        "type": "nested",
        "properties": {
          "id": {
            "type": "keyword"
          },
          "name": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "data": {
            "properties": {
              "bio": {
                "type": "text"
              },
              "twitter": {
                "type": "keyword"
              },
              "linkedin": {
                "type": "keyword",
                "index": false
              },
              "bluesky": {
                "type": "keyword"
              },
              "residence": {
                "type": "keyword"
              },
              "zip-code": {
                "type": "keyword"
              },
              "pictureId": {
                "type": "keyword",
                "index": false
              },
              "emailAlias": {
                "type": "keyword"
              },
              "speakerAlias": {
                "type": "keyword"
              }
            }
          }
        }
      }
    }
  }
}`

// TalkPublicIndexMapping defines the Elasticsearch mapping for the public talks index.
// This mapping excludes sensitive fields like program committee feedback,
// submitter emails, internal notes, and other private data.
const TalkPublicIndexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "default": {
          "type": "standard"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": {
        "type": "keyword"
      },
      "conferenceId": {
        "type": "keyword"
      },
      "conferenceSlug": {
        "type": "keyword"
      },
      "conferenceName": {
        "type": "text",
        "fields": {
          "keyword": {
            "type": "keyword",
            "ignore_above": 256
          }
        }
      },
      "status": {
        "type": "keyword"
      },
      "lastUpdated": {
        "type": "date",
        "format": "strict_date_optional_time||epoch_millis"
      },
      "data": {
        "properties": {
          "title": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "abstract": {
            "type": "text"
          },
          "intendedAudience": {
            "type": "text"
          },
          "format": {
            "type": "keyword"
          },
          "language": {
            "type": "keyword"
          },
          "length": {
            "type": "keyword"
          },
          "level": {
            "type": "keyword"
          },
          "keywords": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "suggestedKeywords": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "suggestedCategory": {
            "type": "keyword"
          },
          "room": {
            "type": "keyword"
          },
          "startTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "endTime": {
            "type": "date",
            "format": "strict_date_optional_time||epoch_millis"
          },
          "video": {
            "type": "keyword",
            "index": false
          },
          "slug": {
            "type": "keyword"
          },
          "published": {
            "type": "keyword"
          },
          "workshopPrerequisites": {
            "type": "text"
          },
          "feedback": {
            "properties": {
              "count": {
                "type": "integer"
              },
              "enjoySum": {
                "type": "integer"
              },
              "usefulSum": {
                "type": "integer"
              },
              "commentList": {
                "type": "text"
              }
            }
          }
        }
      },
      "speakers": {
        "type": "nested",
        "properties": {
          "id": {
            "type": "keyword"
          },
          "name": {
            "type": "text",
            "fields": {
              "keyword": {
                "type": "keyword",
                "ignore_above": 256
              }
            }
          },
          "data": {
            "properties": {
              "bio": {
                "type": "text"
              },
              "twitter": {
                "type": "keyword"
              },
              "linkedin": {
                "type": "keyword",
                "index": false
              },
              "bluesky": {
                "type": "keyword"
              },
              "pictureId": {
                "type": "keyword",
                "index": false
              }
            }
          }
        }
      }
    }
  }
}`
