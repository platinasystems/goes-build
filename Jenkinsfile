#!groovy

import groovy.transform.Field

@Field String email_to = 'sw@platinasystems.com'
@Field String email_from = 'jenkins-bot@platinasystems.com'
@Field String email_reply_to = 'no-reply@platinasystems.com'

pipeline {
    agent any
    stages {
	stage('Checkout') {
	    steps {
		echo "Running build #${env.BUILD_ID} on ${env.JENKINS_URL}"
		dir('goes-boot') {
		    git([
			url: 'git@github.com:platinasystems/goes-boot.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('goes-bmc') {
		    git([
			url: 'git@github.com:platinasystems/goes-bmc.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('goes-example') {
		    git([
			url: 'git@github.com:platinasystems/goes-example.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('goes-build') {
		    git([
			url: 'git@github.com:platinasystems/goes-build.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('goes-platina-mk1') {
		    git([
			url: 'git@github.com:platinasystems/goes-platina-mk1.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('vnet-platina-mk1') {
		    git([
			url: 'git@github.com:platinasystems/vnet-platina-mk1.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('go') {
		    git([
			url: 'git@github.com:platinasystems/go.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('coreboot') {
		    git([
			url: 'git@github.com:platinasystems/coreboot.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('linux') {
		    git([
			url: 'git@github.com:platinasystems/linux.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
			branch: 'master'
			])
		}
		dir('u-boot') {
		    git([
			url: 'git@github.com:platinasystems/u-boot.git',
			credentialsId: "570701f7-c819-4db2-bd31-a0da8a452b41",
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
			sh 'git config --global url.git@github.com:.insteadOf \"https://github.com/\"'
			echo "Building goes..."
			sh 'export PATH=/usr/local/go/bin:/usr/local/x-tools/arm-unknown-linux-gnueabi/bin:${PATH}; go build -v && ./goes-build -x -v -z'
		    }
		}
	    }
	}
    }

    post {
	success {
	    mail body: "GOES build ok: ${env.BUILD_URL}\n\ngoes-platina-mk1-installer is stored on platina4 at /home/jenkins/workspace/go/src/github.com/platinasystems/go/goes-platina-mk1\neg.\nscp 172.16.2.23:/home/jenkins/workspace/go/src/github.com/platinasystems/go/goes-platina-mk1 ~/path/to/somewhere/",
		from: email_from,
		replyTo: email_reply_to,
		subject: 'GOES build ok',
		to: email_to
	}

	failure {
		cleanWs()
		mail body: "GOES build error: ${env.BUILD_URL}",
		from: email_from,
		replyTo: email_reply_to,
		subject: 'GOES BUILD FAILED',
		to: email_to
	}
    }
}
