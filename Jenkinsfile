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
	stage('Checkout') {
	    steps {
		echo "Running build #${env.BUILD_ID} on ${env.JENKINS_URL} GOPATH ${GOPATH}"
		dir('goes-boot') {
		    git([
			url: 'https://github.com/platinasystems/goes-boot.git',
			branch: 'master'
		    ])
		}
		dir('goes-bmc') {
		    git([
			url: 'https://github.com/platinasystems/goes-bmc.git',
			branch: 'master'
		    ])
		}
		dir('goes-example') {
		    git([
			url: 'https://github.com/platinasystems/goes-example.git',
			branch: 'master'
		    ])
		}
		dir('goes-build') {
		    git([
			url: 'https://github.com/platinasystems/goes-build.git',
			branch: 'master'
		    ])
		}
		dir('goes-platina-mk1') {
		    git([
			url: 'https://github.com/platinasystems/goes-platina-mk1.git',
			branch: 'master'
		    ])
		}
		dir('vnet-platina-mk1') {
		    git([
			url: 'https://github.com/platinasystems/vnet-platina-mk1.git',
			branch: 'master'
		    ])
		}
		dir('coreboot') {
		    checkout([$class: 'GitSCM',
			      branches: [[name: '*/master']],
			      doGenerateSubmoduleConfigurations: false,
			      extensions: [[$class: 'SubmoduleOption',
					    disableSubmodules: false,
					    parentCredentials: false,
					    recursiveSubmodules: true,
					    reference: '',
					    trackingSubmodules: false]], 
			      submoduleCfg: [], 
			      userRemoteConfigs: [[url: 'https://github.com/platinasystems/coreboot.git']]])
		}
		dir('linux') {
		    checkout([$class: 'GitSCM',
			      branches: [[name: 'master']], 
			      doGenerateSubmoduleConfigurations: false, 
			      extensions: [[$class: 'CloneOption', depth: 300, noTags: false, reference: '', shallow: true, honorRefspec: true]],
			      submoduleCfg: [], 
			      userRemoteConfigs: [[url: 'https://github.com/platinasystems/linux.git']]])
		}
		dir('u-boot') {
		    git([
			url: 'https://github.com/platinasystems/u-boot.git',
			branch: 'master'
		    ])
		}
	    }
	}
	stage('Build') {
	    steps {
		dir('goes-build') {
		    sshagent(credentials: ['570701f7-c819-4db2-bd31-a0da8a452b41']) {
			echo "Updating worktrees"
			sh 'set -x;env;pwd;[ -d worktrees ] && for repo in worktrees/*/*; do echo $repo; [ -d "$repo" ] && (cd $repo;git fetch origin;git reset --hard HEAD;git rebase origin/master);done || true'
			echo "Setting git config"
			sh 'git config --global url.git@github.com:platinasystems/fe1.insteadOf "https://github.com/platinasystems/fe1"'
			sh 'git config --global url.git@github.com:platinasystems/firmware-fe1.insteadOf "https://github.com/platinasystems/firmware-fe1"'
			echo "Building goes..."
			sh 'export PATH=/usr/local/go/bin:/usr/local/x-tools/arm-unknown-linux-gnueabi/bin:${PATH}; go build -v && ./goes-build -x -v -z'
		    }
		}
	    }
	}
    }


    post {
	success {
	    mail body: "GOES-BUILD build ok: ${env.BUILD_URL}\n",
		from: email_from,
		replyTo: email_reply_to,
		subject: 'GOES build ok',
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
