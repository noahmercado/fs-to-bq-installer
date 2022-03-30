# Firestore to BigQuery Extension Installer
The [`Stream Collections to BigQuery`](https://firebase.google.com/products/extensions/firebase-firestore-bigquery-export) extension for Firebase sends realtime, incremental updates from a specified Cloud Firestore Collection to BigQuery. The current implementation of the extension requires it to be installed for every collection that you want to keep in sync. Additionally, the extension does not backload existing data into BigQuery once installed. It will only stream new Firestore operations. There is instead a separate script that must be run in order to backload historical data. For large scale Firestore instances, this multi-step manual process makes the extension pretty much unusable.

This utility will automatically install multiple instances of the extension in parallel, while also taking care of backloading existing Firestore data.  
  
    
## Disclaimer
This is not an officially supported Google product.  
  
   
## How To
This utility can most easily be run as a Docker container within a Cloud Shell instance. Cloud Shell comes pre-installed with Docker, and is already authenticated to GCP.

- In the GCP console, navigate to the project that contains your Firestore instance (the project that is backing your Firebase project)
  
![Alt text](assets/set-project.png?raw=true "Set-Project")  
  
- Open Cloud Shell  
  
![Alt text](assets/activate-cloud-shell.png?raw=true "Cloud-Shell")  
  
- Start the utility container
```bash
docker run -it -v $CLOUDSDK_CONFIG:/home/installer/.config/gcloud noahmercado/fs-to-bq-installer:latest
```

- When the container boots, it will create a service account in your project with the required permissions needed to install the extension (project scoped `editor` permissions to be specific, in case your InfoSec team is asking :slightly_smiling_face:)

- `fs-to-bq-installer --help` will show you all the available options that can be passed through as extension configuration parameters

- `fs-to-bq-installer -include ALL -max-workers 20` will deploy the extension & backload data for all top level collections in your current working project's Firestore DB (using 20 parallel workers). A BigQuery Dataset will be created named `<YOUR_PROJECT'S_NAME>_firestore_export`, with BQ tables matching the collection names. 

- `fs-to-bq-installer -include ALL -exclude users -max-workers 20` will deploy the extension & backload data for all top level collections in your current working project's Firestore DB (using 20 parallel workers) **EXCEPT** for the collection named `users`

- `fs-to-bq-installer -include metadata,users,topics,/users/noahm/posts` will deploy the extension & backload data for the top level collections `metadata, users, and topics` as well as the nested collection `users/noahm/posts` in your current working project's Firestore DB

- Please reference the official [extension.yml](https://github.com/firebase/extensions/blob/master/firestore-bigquery-export/extension.yaml) manifest to understand each individual parameter 

- Once your installs and data loading workers have completed, running `cleanup` will remove all service account keys for the service account


## Usage
```bash
Usage of fs-to-bq-installer:
  --wildcard-ids
        Creates a column containing a JSON object of all wildcard ids from a documents path.
  -backup-collection string
        This (optional) parameter will allow you to specify a collection for which failed BigQuery updates will be written to.
  -clustering data,document_id,timestamp
        This parameter will allow you to set up Clustering for the BigQuery Table created by the extension. (for example: data,document_id,timestamp- no whitespaces). You can select up to 4 comma separated fields(order matters). Available schema extensions table fields for clustering: `document_id, timestamp, event_id, operation, data`.
  -dataset-id string
        The ID of the BigQuery dataset (default "<YOUR_PROJECT_ID>_firestore_export")
  -dataset-location string
        Where do you want to deploy the BigQuery dataset created for this extension? For help selecting a location, refer to the [location selection guide](https://cloud.google.com/bigquery/docs/locations). (default "us")
  -exclude string
        A comma separated list of collection names to exclude when include is set to 'ALL'
  -include string
        A comma separated list of collection names to include. Set to 'ALL' to include all collections
  -location string
        Where do you want to deploy the functions created for this extension?  You usually want a location close to your database. For help selecting a location, refer to the [location selection guide](https://firebase.google.com/docs/functions/locations). (default "us-central1")
  -max-workers int
        The maximum number of works to run in parallel when deploying the extension (default 5)
  -project-id string
        The GCP Project ID (default: will look for a local `.firebaserc` file, followed by env var `$GOOGLE_PROJECT_ID`. If neither are found is empty string)
  -table-partitioning string
        This parameter will allow you to partition the BigQuery table and BigQuery view created by the extension based on data ingestion time. You may select the granularity of partitioning based upon one of: HOUR, DAY, MONTH, YEAR. This will      generate one partition per day, hour, month or year, respectively. (default "NONE")
  -time-partitioning-field timestamp
        BigQuery table column/schema field name for TimePartitioning. You can choose schema available as timestamp OR new custom defined column that will be assigned to the selected Firestore Document field below. Defaults to pseudo column _PARTITIONTIME if unspecified. Cannot be changed if Table is already partitioned.
  -time-partitioning-field-type string
        Parameter for BigQuery SQL schema field type for the selected Time Partitioning Firestore Document field option. Cannot be changed if Table is already partitioned. (default "omit")
  -time-partitioning-firestore-field postDate
        This parameter will allow you to partition the BigQuery table  created by the extension based on selected. The Firestore Document field value must be a top-level TIMESTAMP, DATETIME, DATE field BigQuery string format or Firestore timestamp(will be converted to BigQuery TIMESTAMP). Cannot be changed if Table is already partitioned. example: postDate
  -transform-function-url string
        Specify a function URL to call that will transform the payload that will be written to BigQuery. See the pre-install documentation for more details.

```  
  
<!-- (default: will look for a local `.firebaserc` file, followed by env var `$GOOGLE_PROJECT_ID`. If neither are found is empty string) -->
### TODO:
- Implement `-recursive` flag logic
- Build for multiple archs
- Cleanup docs
