1. There seems to be no way to list repos in a project
2. There is a lot of confusion around share/attach/move repos re: project
3. There is no API to create a repo and assign it to a project, but it appears that,
by convention, the project name is pre-pending to the repo name and this works (but only for repos created in a project)
4. Only admin auth token (bearer) is supported. This will present some compatibility problems for current TF users as
if they include projects in their TF, we must fail them if they don't have bearer. Or, we have to remove support for anything else.
   In addition, the bearer token can't be scoped to just projects. A potential security issue
5. Roles are both global and project scoped. 
6. XRAY permissions are global even if created within the scope of a project - The requirement to scope xray permissions to a project can't be met
7. You can't list the default roles for a project until the project is created, meaning it can't be treated as an independent resource
  Global roles do not apply and are not accessible. 
8. Data validation is sketchy. If you supply an invalid role for a project, it doesn't error, but it also doesn't show up on the UI
   
Because there is a lack of resource definition, a 'virtual' resource will likely need to be created in TF dsl; though this is not advisable. 
The lack of clear resource management for projects will significantly complicate and slow down the Terraform application 
as numerous calls to various endpoints will need to be called to construct a virtual entity.  