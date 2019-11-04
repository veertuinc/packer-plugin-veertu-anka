

pipeline {
    agent { node { label 'build-packer' }} 
    stages {
        stage('Build') {
            steps {
                sh 'FILES=$(ls) && mkdir -p src/github.com/veertuinc/packer-builder-veertu-anka && for FILE in "${FILES[@]}"; do mv $FILE ./src/github.com/veertuinc/packer-builder-veertu-anka/  ; done'
                sh 'export GOPATH=$PWD && cd ./src/github.com/veertuinc/packer-builder-veertu-anka/ && make build'
            }
        }
        stage('Publish') {
            steps {
                archiveArtifacts artifacts: '**/src/github.com/veertuinc/packer-builder-veertu-anka/packer-builder-veertu-anka'
            }
        }
    }
}


