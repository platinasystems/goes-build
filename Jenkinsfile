#!groovy

import groovy.transform.Field

@Field String email_to = 'sw@platinasystems.com'
@Field String email_from = 'jenkins-bot@platinasystems.com'
@Field String email_reply_to = 'no-reply@platinasystems.com'

pipeline {
    agent any
    environment {
	GOPATH = "$WORKSPACE/go-pkg"
	HOME = "$WORKSPACE"
    }
    stages {
	stage('Build') {
	    steps {
		sshagent(credentials: ['570701f7-c819-4db2-bd31-a0da8a452b41']) {
		    echo "Running build #${env.BUILD_ID} branch ${env.BRANCH_NAME} on ${env.JENKINS_URL} GOPATH ${GOPATH}"
		    sh 'make bindeb-pkg'
		}
	    }
	}
    }

    post {
	success {
	    archiveArtifacts artifacts: '../*.deb,../*.changes,../*.buildinfo'
	    mail body: "GOES-BUILD build ok: ${env.BUILD_URL}\n",
		from: email_from,
		replyTo: email_reply_to,
		subject: 'GOES-BUILD build ok',
		to: email_to
	}
	failure {
	    cleanWs()
	    mail body: "GOES-BUILD build error: ${env.BUILD_URL}",
		from: email_from,
		replyTo: email_reply_to,
		subject: 'GOES-BUILD BUILD FAILED',
		to: email_to
	}
    }
}
