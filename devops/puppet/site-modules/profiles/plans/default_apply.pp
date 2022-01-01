plan profiles::default_apply(
     TargetSpec $targets,
   ) {

     # Install the puppet-agent package if Puppet is not detected.
     # Copy over custom facts from the Bolt modulepath.
     # Run the `facter` command line tool to gather target information.
     $targets.apply_prep

     # Compile the manifest block into a catalog
     apply($targets) {

       include 'kite_base'
       include regsubst($facts['node_name'], '-', '_', 'G')
     }
}
