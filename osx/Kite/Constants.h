//
//  Constants.h
//  Kite
//
//  Created by Tarak Upadhyaya on 5/17/17.
//  Copyright © 2017 Manhattan Engineering. All rights reserved.
//

#ifndef Constants_h
#define Constants_h

#ifdef ENTERPRISE
NSString *const CONFIGURATION=@"enterprise";
#elif DEBUG
NSString *const CONFIGURATION=@"debug";
#else
NSString *const CONFIGURATION=@"release";
#endif

#endif /* Constants_h */
