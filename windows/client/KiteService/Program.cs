using System;
using System.Collections.Generic;
using System.ServiceProcess;
using System.Text;

namespace KiteService {
    
    public static class Program {

        public static void Main() {
            ServiceBase[] ServicesToRun;
            ServicesToRun = new ServiceBase[]  { 
				new KiteService() 
			};
            ServiceBase.Run(ServicesToRun);
        }
    }
}
