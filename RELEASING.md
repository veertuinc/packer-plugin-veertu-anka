# Releasing

All steps for releasing are internal to Veertu and are not public.

1. Create version branch with `release/v<version>` for both github-repo-packer-plugin-veertu-anka and internal anka-packer repo
2. Update VERSION file to `<version>` inside github-repo-packer-plugin-veertu-anka
3. Commit and push both branches
4. Make your changes
5. Run build job in Jenkins
6. Run test job in Jenkins
7. If tests pass, merge version branch into master (both repos)
8. Run release job in Jenkins (master branch)